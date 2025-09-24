package internal

import (
	"sync"
	"time"
)

// --- Constants for Test Statuses ---
const (
	StatusRun     = "run"
	StatusPass    = "pass"
	StatusFail    = "fail"
	StatusSkip    = "skip"
	StatusOutput  = "output"
	StatusRunning = "running" // Internal status for tests currently executing
)

// --- TestEvent (External representation from `go test -json`) ---
type TestEvent struct {
	Time    time.Time `json:"Time"`
	Action  string    `json:"Action"`
	Package string    `json:"Package"`
	Test    string    `json:"Test"` // Empty for package-level events
	Output  string    `json:"Output"`
	Elapsed float64   `json:"Elapsed"` // In seconds
}

// --- TestResult (Internal aggregated representation for a single test) ---
type TestResult struct {
	Name     string // Full test name, e.g., TestMyFunction
	Package  string
	Status   string // StatusPass, StatusFail, StatusSkip, StatusRunning
	Duration time.Duration
	Output   []string // Raw output from the test
}

// --- PackageResults (Aggregated results for a single package) ---
type PackageResults struct {
	Name     string
	Tests    []TestResult
	Status   string // Derived: StatusPass, StatusFail, StatusSkip
	Total    int
	Passed   int
	Failed   int
	Skipped  int
	Duration time.Duration // Sum of individual test durations in the package
}

// --- TestSummary (Overall results of the entire test run) ---
type TestSummary struct {
	sync.Mutex              // Protects global counters
	Results    []TestResult // Flat list of all individual test results
	Passed     int
	Failed     int
	Skipped    int
	Total      int
}
