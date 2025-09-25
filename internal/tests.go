package internal

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

func Execute(args []string) {
	cmdArgs := []string{"test", "-json"}
	userArgs := os.Args[1:]

	if len(userArgs) == 0 {
		cmdArgs = append(cmdArgs, "./...")
	} else {
		cmdArgs = append(cmdArgs, userArgs...)
	}

	summary, err := runTests(cmdArgs)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"%s\n",
			errorStyle.Render(fmt.Sprintf("Error running tests: %v", err)),
		)
		os.Exit(1)
	}

	// Capture all display output as a single string and wrap it
	fullOutput := displayResults(summary)
	fmt.Println(AppOverallOutputStyle.Render(fullOutput))

	if summary.Failed > 0 {
		os.Exit(1)
	}
	os.Exit(0)
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

func processEvent(event *TestEvent, testMap map[string]*TestResult, summary *TestSummary) {
	if event.Test == "" {
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

	switch event.Action {
	case StatusOutput:
		output := strings.TrimSpace(event.Output)
		if output != "" {
			result.Output = append(result.Output, output)
		}

	case StatusPass, StatusFail, StatusSkip:
		result.Status = event.Action
		result.Duration = time.Duration(event.Elapsed * float64(time.Second))

		switch event.Action {
		case StatusPass:
			summary.Passed++
		case StatusFail:
			summary.Failed++
		case StatusSkip:
			summary.Skipped++
		}
	}
}
