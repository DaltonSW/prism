package internal

import (
	"testing"
	"time"
)

func TestParseBenchmarkLine(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantOK  bool
		base    string
		threads int
		iter    int
	}{
		{name: "valid single thread", input: "BenchmarkFoo\t1000\t\t500 ns/op\t0 B/op\t0 allocs/op", wantOK: true, base: "BenchmarkFoo", threads: 0, iter: 1000},
		{name: "valid with threads", input: "BenchmarkBar-4\t5000\t\t250 ns/op\t16 B/op\t1 allocs/op", wantOK: true, base: "BenchmarkBar", threads: 4, iter: 5000},
		{name: "missing tab", input: "BenchmarkFoo 1000 500 ns/op", wantOK: false, base: "", threads: 0, iter: 0},
		{name: "wrong prefix", input: "TestFoo\t1000\t500 ns/op", wantOK: false, base: "", threads: 0, iter: 0},
		{name: "empty", input: "", wantOK: false, base: "", threads: 0, iter: 0},
		{name: "non numeric iter", input: "BenchmarkFoo\tabc\t500 ns/op", wantOK: true, base: "BenchmarkFoo", threads: 0, iter: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseBenchmarkLine(tc.input)
			if ok != tc.wantOK {
				t.Fatalf("parseBenchmarkLine(%q) ok = %v, want %v", tc.input, ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if got.BaseName != tc.base {
				t.Errorf("BaseName = %q, want %q", got.BaseName, tc.base)
			}
			if got.Threads != tc.threads {
				t.Errorf("Threads = %d, want %d", got.Threads, tc.threads)
			}
			if got.Iterations != tc.iter {
				t.Errorf("Iterations = %d, want %d", got.Iterations, tc.iter)
			}
		})
	}
}

func TestSplitBenchmarkName(t *testing.T) {
	cases := []struct {
		in     string
		base   string
		thread int
	}{
		{"BenchmarkFoo", "BenchmarkFoo", 0},
		{"BenchmarkFoo-1", "BenchmarkFoo", 1},
		{"BenchmarkFoo-16", "BenchmarkFoo", 16},
		{"BenchmarkFoo-x", "BenchmarkFoo-x", 0},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			base, thread := splitBenchmarkName(tc.in)
			if base != tc.base || thread != tc.thread {
				t.Errorf("splitBenchmarkName(%q) = (%q, %d), want (%q, %d)", tc.in, base, thread, tc.base, tc.thread)
			}
		})
	}
}

func TestBenchmarkFormatters(t *testing.T) {
	if got := formatIterations(0); got != "—" {
		t.Errorf("formatIterations(0) = %q, want %q", got, "—")
	}
	if got := formatIterations(1000); got != "1000" {
		t.Errorf("formatIterations(1000) = %q, want %q", got, "1000")
	}

	if got := formatThreads(0); got != "—" {
		t.Errorf("formatThreads(0) = %q, want %q", got, "—")
	}
	if got := formatThreads(8); got != "8" {
		t.Errorf("formatThreads(8) = %q, want %q", got, "8")
	}

	if got := formatBenchmarkDuration(0); got != "—" {
		t.Errorf("formatBenchmarkDuration(0) = %q, want %q", got, "—")
	}
	if got := formatBenchmarkDuration(250 * time.Millisecond); got != "250ms" {
		t.Errorf("formatBenchmarkDuration(250ms) = %q, want %q", got, "250ms")
	}
}

func TestCollectBenchmarkMetricKeys_PreferredOrder(t *testing.T) {
	results := []*BenchmarkResult{
		{Metrics: map[string]string{"allocs/op": "0", "MB/s": "10", "ns/op": "500"}},
		{Metrics: map[string]string{"B/op": "16", "custom/op": "1"}},
	}

	keys := collectBenchmarkMetricKeys(results)
	if len(keys) != 5 {
		t.Fatalf("expected 5 keys, got %d (%v)", len(keys), keys)
	}

	wantOrder := []string{"ns/op", "B/op", "allocs/op", "MB/s", "custom/op"}
	for i, want := range wantOrder {
		if keys[i] != want {
			t.Errorf("key[%d] = %q, want %q (full slice: %v)", i, keys[i], want, keys)
		}
	}
}

func TestBenchmarksHaveThreads(t *testing.T) {
	if benchmarksHaveThreads([]*BenchmarkResult{{Threads: 0}, {Threads: 0}}) {
		t.Error("expected false when no benchmarks have threads")
	}
	if !benchmarksHaveThreads([]*BenchmarkResult{{Threads: 0}, {Threads: 2}}) {
		t.Error("expected true when any benchmark has threads")
	}
	if benchmarksHaveThreads(nil) {
		t.Error("expected false for empty slice")
	}
}

func TestFilterBenchmarkResults_OnlyFails(t *testing.T) {
	prev := GlobalConfig.OnlyFails
	defer func() { GlobalConfig.OnlyFails = prev }()

	results := []*BenchmarkResult{
		{Name: "A", Status: StatusPass},
		{Name: "B", Status: StatusFail},
		{Name: "C", Status: StatusSkip},
	}

	GlobalConfig.OnlyFails = false
	if got := len(filterBenchmarkResults(results)); got != 3 {
		t.Errorf("with OnlyFails=false, expected 3 results, got %d", got)
	}

	GlobalConfig.OnlyFails = true
	got := filterBenchmarkResults(results)
	if len(got) != 1 || got[0].Name != "B" {
		t.Errorf("with OnlyFails=true, expected only failing benchmark, got %v", got)
	}
}

func TestProcessBenchmarkEvent_PassAndMetrics(t *testing.T) {
	summary := &BenchmarkSummary{
		PackageEnv: make(map[string]*BenchmarkPackageEnv),
	}
	benchmarkMap := make(map[string]*BenchmarkResult)

	processBenchmarkEvent(&TestEvent{Action: "run", Package: "p", Test: "BenchmarkFoo"}, benchmarkMap, summary)
	processBenchmarkEvent(&TestEvent{Action: "output", Package: "p", Test: "BenchmarkFoo", Output: "BenchmarkFoo\t1000\t500 ns/op\t0 B/op"}, benchmarkMap, summary)
	processBenchmarkEvent(&TestEvent{Action: "pass", Package: "p", Test: "BenchmarkFoo", Elapsed: 0.5}, benchmarkMap, summary)

	result := benchmarkMap["p/BenchmarkFoo"]
	if result == nil {
		t.Fatal("expected benchmark result to be recorded")
	}
	if result.Status != StatusPass {
		t.Errorf("expected Status=Pass, got %v", result.Status)
	}
	if result.Iterations != 1000 {
		t.Errorf("expected Iterations=1000, got %d", result.Iterations)
	}
	if result.Metrics["ns/op"] != "500" {
		t.Errorf("expected ns/op=500, got %q", result.Metrics["ns/op"])
	}
	if result.Metrics["B/op"] != "0" {
		t.Errorf("expected B/op=0, got %q", result.Metrics["B/op"])
	}
	if summary.Succeeded != 1 {
		t.Errorf("expected summary.Succeeded=1, got %d", summary.Succeeded)
	}
}

func TestProcessBenchmarkEvent_FailAndSkip(t *testing.T) {
	summary := &BenchmarkSummary{
		PackageEnv: make(map[string]*BenchmarkPackageEnv),
	}
	benchmarkMap := make(map[string]*BenchmarkResult)

	processBenchmarkEvent(&TestEvent{Action: "fail", Package: "p", Test: "BenchmarkFail", Elapsed: 0.1}, benchmarkMap, summary)
	processBenchmarkEvent(&TestEvent{Action: "skip", Package: "p", Test: "BenchmarkSkip"}, benchmarkMap, summary)

	if summary.Failed != 1 {
		t.Errorf("expected Failed=1, got %d", summary.Failed)
	}
	if summary.Skipped != 1 {
		t.Errorf("expected Skipped=1, got %d", summary.Skipped)
	}
}

func TestUpdateBenchmarkPackageMetadata(t *testing.T) {
	summary := &BenchmarkSummary{
		PackageEnv: make(map[string]*BenchmarkPackageEnv),
	}

	updateBenchmarkPackageMetadata(summary, "p", "goos: linux")
	updateBenchmarkPackageMetadata(summary, "p", "goarch: amd64")
	updateBenchmarkPackageMetadata(summary, "p", "cpu: Intel(R) Core(TM)")
	updateBenchmarkPackageMetadata(summary, "p", "")
	updateBenchmarkPackageMetadata(summary, "", "goos: ignored")
	updateBenchmarkPackageMetadata(summary, "p", "malformed line without colon")

	env, ok := summary.PackageEnv["p"]
	if !ok {
		t.Fatal("expected env for package p")
	}
	if env.Goos != "linux" {
		t.Errorf("expected Goos=linux, got %q", env.Goos)
	}
	if env.Goarch != "amd64" {
		t.Errorf("expected Goarch=amd64, got %q", env.Goarch)
	}
	if env.CPU != "Intel(R) Core(TM)" {
		t.Errorf("expected CPU=Intel(R) Core(TM), got %q", env.CPU)
	}
}
