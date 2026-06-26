package internal

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/yarlson/pin"
)

var coverProfilePath string

func Execute(args []string) {
	benchMode := containsBenchmarkFlag(args)

	cmdArgs := []string{"test", "-json"}

	if len(args) == 0 {
		cmdArgs = append(cmdArgs, "./...")
	} else {
		cmdArgs = append(cmdArgs, args...)
	}

	if GlobalConfig.Cover {
		cmdArgs = append(cmdArgs, "-cover")
		coverProfilePath = filepath.Join(os.TempDir(), fmt.Sprintf("prism-cover-%d.out", os.Getpid()))
		cmdArgs = append(cmdArgs, "-coverprofile="+coverProfilePath)
	}
	if GlobalConfig.Count != "" {
		cmdArgs = append(cmdArgs, "-count="+GlobalConfig.Count)
	}
	if GlobalConfig.Filter != "" {
		cmdArgs = append(cmdArgs, "-run", GlobalConfig.Filter)
	}

	if benchMode {
		p := pin.New(" Running benchmarks...",
			pin.WithPosition(pin.PositionRight),
			pin.WithTextColor(pin.ColorCyan),
			pin.WithSpinnerColor(pin.ColorMagenta),
		)

		cancel := p.Start(context.Background())
		defer cancel()

		summary, err := runBenchmarks(cmdArgs)

		p.Stop("")

		if err != nil {
			fmt.Fprintf(
				os.Stderr,
				"%s\n",
				errorStyle.Render(fmt.Sprintf("Error running benchmarks: %v", err)),
			)
			os.Exit(1)
		}

		if summary.Total == 0 {
			displayZeroBenchmarks()
		} else {
			displayBenchmarkResults(summary)
		}
	} else {
		p := pin.New(" Running tests...",
			pin.WithPosition(pin.PositionRight),
			pin.WithTextColor(pin.ColorCyan),
			pin.WithSpinnerColor(pin.ColorMagenta),
		)

		cancel := p.Start(context.Background())
		defer cancel()

		summary, err := runTests(cmdArgs)

		p.Stop("")

		if err != nil {
			fmt.Fprintf(
				os.Stderr,
				"%s\n",
				errorStyle.Render(fmt.Sprintf("Error running tests: %v", err)),
			)
			os.Exit(1)
		}

		if GlobalConfig.Cover && coverProfilePath != "" {
			if funcCov, totalPct, loadErr := loadCoverage(coverProfilePath); loadErr == nil {
				summary.PackageFuncCoverage = funcCov
				summary.TotalCoverage = totalPct
			}
		}

		// Capture all display output as a single string and wrap it
		if summary.Total == 0 {
			displayZeroState()
		} else {
			displayResults(summary)
		}
	}

}

func containsBenchmarkFlag(args []string) bool {
	for i := range len(args) {
		arg := args[i]
		if strings.HasPrefix(arg, "-bench") || strings.HasPrefix(arg, "-test.bench") {
			return true
		}
	}
	return false
}

func runTests(args []string) (*TestSummary, error) {
	cmd := exec.CommandContext(context.Background(), "go", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	summary := &TestSummary{
		Results: make([]TestResult, 0),
	}
	testMap := make(map[string]*TestResult)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			var event TestEvent
			if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
				fmt.Fprintf(
					os.Stderr,
					"%s\n",
					errorStyle.Render(fmt.Sprintf(
						"Warning: Failed to unmarshal JSON event: %v (line: %s)",
						err,
						scanner.Text(),
					)),
				)
				continue
			}
			processEvent(&event, testMap, summary)
		}
		if err := scanner.Err(); err != nil && err != io.EOF {
			fmt.Fprintf(
				os.Stderr,
				"%s\n",
				errorStyle.Render(fmt.Sprintf("Error reading stdout: %v", err)),
			)
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Fprintf(os.Stderr, "%s\n", scanner.Text())
		}
		if err := scanner.Err(); err != nil && err != io.EOF {
			fmt.Fprintf(
				os.Stderr,
				"%s\n",
				errorStyle.Render(fmt.Sprintf("Error reading stderr: %v", err)),
			)
		}
	}()

	cmdErr := cmd.Wait()

	wg.Wait()

	for _, result := range testMap {
		summary.Results = append(summary.Results, *result)
	}

	if cmdErr != nil {
		if exitErr, ok := cmdErr.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return summary, nil
			}
			return nil, fmt.Errorf(
				"command exited with non-zero status %d: %w",
				exitErr.ExitCode(),
				cmdErr,
			)
		}
		return nil, fmt.Errorf("command execution failed: %w", cmdErr)
	}

	return summary, nil
}

func runBenchmarks(args []string) (*BenchmarkSummary, error) {
	cmd := exec.CommandContext(context.Background(), "go", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	summary := &BenchmarkSummary{
		Results:    make([]*BenchmarkResult, 0),
		PackageEnv: make(map[string]*BenchmarkPackageEnv),
	}
	benchmarkMap := make(map[string]*BenchmarkResult)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			var event TestEvent
			if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
				fmt.Fprintf(
					os.Stderr,
					"%s\n",
					errorStyle.Render(fmt.Sprintf(
						"Warning: Failed to unmarshal JSON event: %v (line: %s)",
						err,
						scanner.Text(),
					)),
				)
				continue
			}
			processBenchmarkEvent(&event, benchmarkMap, summary)
		}
		if err := scanner.Err(); err != nil && err != io.EOF {
			fmt.Fprintf(
				os.Stderr,
				"%s\n",
				errorStyle.Render(fmt.Sprintf("Error reading stdout: %v", err)),
			)
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Fprintf(os.Stderr, "%s\n", scanner.Text())
		}
		if err := scanner.Err(); err != nil && err != io.EOF {
			fmt.Fprintf(
				os.Stderr,
				"%s\n",
				errorStyle.Render(fmt.Sprintf("Error reading stderr: %v", err)),
			)
		}
	}()

	cmdErr := cmd.Wait()

	wg.Wait()

	if cmdErr != nil {
		if exitErr, ok := cmdErr.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return summary, nil
			}
			return nil, fmt.Errorf(
				"command exited with non-zero status %d: %w",
				exitErr.ExitCode(),
				cmdErr,
			)
		}
		return nil, fmt.Errorf("command execution failed: %w", cmdErr)
	}

	return summary, nil
}

func processEvent(event *TestEvent, testMap map[string]*TestResult, summary *TestSummary) {
	if event.Test == "" {
		if event.Action == "output" {
			line := strings.TrimSpace(event.Output)
			if strings.HasPrefix(line, "coverage: ") {
				summary.Lock()
				if summary.PackageCoverage == nil {
					summary.PackageCoverage = make(map[string]string)
				}
				summary.PackageCoverage[event.Package] = line
				summary.Unlock()
			}
		}
		return
	}

	key := event.Package + "/" + event.Test

	summary.Lock()
	defer summary.Unlock()

	result, exists := testMap[key]
	if !exists {
		result = &TestResult{
			Name:    event.Test,
			Package: event.Package,
			Status:  StatusRunning,
			Output:  make([]string, 0),
		}
		testMap[key] = result
		summary.Total++
	}

	switch action := Status(event.Action); action {
	case StatusOutput:
		output := strings.TrimSpace(event.Output)
		if output != "" {
			result.Output = append(result.Output, output)
		}

	case StatusPass, StatusFail, StatusSkip:
		result.Status = action
		result.Duration = time.Duration(event.Elapsed * float64(time.Second))

		switch action {
		case StatusPass:
			summary.Passed++
		case StatusFail:
			summary.Failed++
		case StatusSkip:
			summary.Skipped++
		}
	}
}
