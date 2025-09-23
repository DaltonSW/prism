package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss/v2"
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

// --- Lipgloss Styles (Less harsh colors) ---
var (
	passStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#8AFF8A")).Bold(true) // Light Green
	failStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8A80")).Bold(true) // Light Red/Coral
	skipStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFAA")).Bold(true) // Pale Yellow

	packageStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#8AFFFB")).Bold(true) // Light Aqua
	testNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#E0E0E0"))            // Off-white
	durationStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A0A0A0"))            // Medium Gray
	// Adjusted MarginLeft to align failure output with the start of the 'TEST NAME' column.
	outputStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#C0C0C0")).Italic(true).MarginLeft(9)

	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8AFF")).Bold(true).Underline(true) // Soft Magenta

	summaryBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			MarginTop(1).
			BorderForeground(lipgloss.Color("#6A6A6A")) // Darker Gray for border

	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4500")).Bold(true) // Orange-Red

	packageSeparatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Faint(true) // Dark Gray

	// New style for table headers (within package)
	tableHeaderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Bold(true).Underline(true) // Subtle gray underline
)

// getStatusInfo is a helper to return the correct icon and style for a given status.
func getStatusInfo(status string) (icon string, style lipgloss.Style) {
	switch status {
	case StatusPass:
		return "✓", passStyle
	case StatusFail:
		return "✗", failStyle
	case StatusSkip:
		return "⊝", skipStyle
	default: // StatusRunning or unknown
		return "◌", lipgloss.NewStyle().Foreground(lipgloss.Color("#B0B0B0")) // Gray dot
	}
}

func old_main() {
	// Construct args for 'go test'. Always include -json.
	cmdArgs := []string{"test", "-json"}
	userArgs := os.Args[1:]

	// Append user-provided arguments. If none, default to "./..."
	if len(userArgs) == 0 {
		cmdArgs = append(cmdArgs, "./...")
	} else {
		cmdArgs = append(cmdArgs, userArgs...)
	}

	summary, err := runTests(cmdArgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", errorStyle.Render(fmt.Sprintf("Error running tests: %v", err)))
		os.Exit(1)
	}

	displayResults(summary)

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
	// Use a map to aggregate individual TestResults keyed by unique identifier (package/test)
	testMap := make(map[string]*TestResult)

	var wg sync.WaitGroup
	wg.Add(1) // For the parser goroutine

	// Goroutine to read stdout and process events
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			var event TestEvent
			if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
				// Log JSON unmarshal errors, but try to continue
				fmt.Fprintf(os.Stderr, "%s\n", errorStyle.Render(
					fmt.Sprintf("Warning: Failed to unmarshal JSON event: %v (line: %s)", err, scanner.Text())))
				continue
			}
			processEvent(&event, testMap, summary)
		}
		if err := scanner.Err(); err != nil && err != io.EOF {
			fmt.Fprintf(os.Stderr, "%s\n", errorStyle.Render(fmt.Sprintf("Error reading stdout: %v", err)))
		}
	}()

	// Stream stderr directly to os.Stderr, potentially with styling
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Fprintf(os.Stderr, "%s\n", scanner.Text())
		}
		if err := scanner.Err(); err != nil && err != io.EOF {
			fmt.Fprintf(os.Stderr, "%s\n", errorStyle.Render(fmt.Sprintf("Error reading stderr: %v", err)))
		}
	}()

	// Wait for `go test` command to complete
	cmdErr := cmd.Wait()

	// Wait for the goroutine parsing stdout to finish
	wg.Wait()

	// After all events are processed, consolidate map to slice
	for _, result := range testMap {
		summary.Results = append(summary.Results, *result)
	}

	// Handle the command exit status
	if cmdErr != nil {
		if exitErr, ok := cmdErr.(*exec.ExitError); ok {
			// `go test` returns exit code 1 if tests failed (but command itself ran)
			if exitErr.ExitCode() == 1 {
				return summary, nil
			}
			return nil, fmt.Errorf("command exited with non-zero status %d: %w", exitErr.ExitCode(), cmdErr)
		}
		return nil, fmt.Errorf("command execution failed: %w", cmdErr)
	}

	return summary, nil
}

// processEvent updates the testMap and the overall summary based on each event.
func processEvent(event *TestEvent, testMap map[string]*TestResult, summary *TestSummary) {
	// Filter out package-level events for this tool's scope (focused on individual tests)
	if event.Test == "" {
		return
	}

	key := event.Package + "/" + event.Test

	// Lock the summary for concurrent updates to global counters
	summary.Lock()
	defer summary.Unlock()

	result, exists := testMap[key]
	if !exists {
		result = &TestResult{
			Name:    event.Test,
			Package: event.Package,
			Status:  StatusRunning, // Initial status
			Output:  make([]string, 0),
		}
		testMap[key] = result
		summary.Total++ // Increment total count when test starts
	}

	switch event.Action {
	case StatusOutput:
		output := strings.TrimSpace(event.Output)
		if output != "" {
			result.Output = append(result.Output, output)
		}

	case StatusPass, StatusFail, StatusSkip:
		// Update final status and duration
		result.Status = event.Action
		result.Duration = time.Duration(event.Elapsed * float64(time.Second))

		// Update global summary counters
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

// displayResults groups tests by package and then displays them.
func displayResults(overallSummary *TestSummary) {
	fmt.Println(headerStyle.Render("Test Results"))
	fmt.Println()

	// 1. Group tests by package
	groupedByPackage := make(map[string]*PackageResults)
	for _, testResult := range overallSummary.Results {
		pkgName := testResult.Package
		if _, ok := groupedByPackage[pkgName]; !ok {
			groupedByPackage[pkgName] = &PackageResults{
				Name:   pkgName,
				Tests:  []TestResult{},
				Status: StatusPass, // Assume pass until a failure
			}
		}
		pkgResults := groupedByPackage[pkgName]
		pkgResults.Tests = append(pkgResults.Tests, testResult)
		pkgResults.Total++
		pkgResults.Duration += testResult.Duration // Summing individual test durations for package duration

		switch testResult.Status {
		case StatusPass:
			pkgResults.Passed++
		case StatusFail:
			pkgResults.Failed++
			pkgResults.Status = StatusFail // If any test fails, the package fails
		case StatusSkip:
			pkgResults.Skipped++
			// If package status is still "pass" and all tests were skipped, set package status to skipped
			if pkgResults.Status == StatusPass && pkgResults.Skipped == pkgResults.Total {
				pkgResults.Status = StatusSkip
			}
		}
	}

	// 2. Sort package names alphabetically for consistent output
	packageNames := make([]string, 0, len(groupedByPackage))
	for pkgName := range groupedByPackage {
		packageNames = append(packageNames, pkgName)
	}
	sort.Strings(packageNames)

	// 3. Iterate and display each package's results
	for _, pkgName := range packageNames {
		pkgResults := groupedByPackage[pkgName]
		displayPackageBlock(pkgResults)
	}

	// 4. Display overall summary
	displayOverallSummary(overallSummary)
}

// TODO: Make the display functionlity less hardcoded. This is definitely pretty gross

// displayPackageBlock handles the display of a single package, including its header and tests in a table.
func displayPackageBlock(pkgResults *PackageResults) {
	icon, statusStyle := getStatusInfo(pkgResults.Status)
	statusText := strings.ToUpper(pkgResults.Status)

	// Package Header Line
	fmt.Printf("%s %s %s %s %s\n",
		statusStyle.Render(icon),
		statusStyle.Render(statusText),
		packageStyle.Render(pkgResults.Name),
		durationStyle.Render(fmt.Sprintf("(%v)", pkgResults.Duration)),
		lipgloss.NewStyle().Faint(true).Render(
			fmt.Sprintf(" (%d total, %s %d passed, %s %d failed, %s %d skipped)",
				pkgResults.Total,
				passStyle.Render(""), pkgResults.Passed,
				failStyle.Render(""), pkgResults.Failed,
				skipStyle.Render(""), pkgResults.Skipped,
			),
		),
	)
	fmt.Println(packageSeparatorStyle.Render(strings.Repeat("─", 80)))

	// Calculate column widths for the table of tests within this package
	const iconColWidth = 2       // e.g., "✓ "
	const statusTextColWidth = 5 // e.g., "PASS "
	const durationColWidth = 9   // e.g., "(123.45s)"
	const columnPadding = 1      // Space between columns

	// Calculate max width for Test Name column (after stripping "Test" prefix)
	maxStrippedTestNameWidth := 0
	for _, testResult := range pkgResults.Tests {
		strippedName := strings.TrimPrefix(testResult.Name, "Test")
		if len(strippedName) > maxStrippedTestNameWidth {
			maxStrippedTestNameWidth = len(strippedName)
		}
	}

	// Cap the test name width to ensure total row doesn't exceed common terminal width (e.g., 80)
	// (Total row width: icon + pad + status + pad + name + pad + duration)
	remainingWidth := 80 - (iconColWidth + columnPadding + statusTextColWidth + columnPadding + durationColWidth + columnPadding)
	if maxStrippedTestNameWidth > remainingWidth {
		maxStrippedTestNameWidth = remainingWidth
	}
	// Ensure a minimum width for the test name, otherwise it might disappear
	if maxStrippedTestNameWidth < 10 {
		maxStrippedTestNameWidth = 10
	}

	// Create reusable lipgloss styles for each column, defining their width and alignment
	iconColStyle := lipgloss.NewStyle().Width(iconColWidth).Align(lipgloss.Left)
	statusColStyle := lipgloss.NewStyle().Width(statusTextColWidth).Align(lipgloss.Left)
	testNameColStyle := lipgloss.NewStyle().Width(maxStrippedTestNameWidth).Align(lipgloss.Left) // Use maxStrippedTestNameWidth here
	durationColStyle := lipgloss.NewStyle().Width(durationColWidth).Align(lipgloss.Right)

	// Sort tests within the package by status (Fail, Skip, Pass) then by name
	sort.Slice(pkgResults.Tests, func(i, j int) bool {
		// Define an order for statuses
		statusOrder := map[string]int{
			StatusFail:    3,
			StatusSkip:    2,
			StatusPass:    1,
			StatusRunning: 0, // Should not appear in final results, but for robustness
		}
		orderI := statusOrder[pkgResults.Tests[i].Status]
		orderJ := statusOrder[pkgResults.Tests[j].Status]

		if orderI != orderJ {
			return orderI < orderJ // Sort by status first
		}
		// If statuses are the same, sort alphabetically by stripped test name
		nameI := strings.TrimPrefix(pkgResults.Tests[i].Name, "Test")
		nameJ := strings.TrimPrefix(pkgResults.Tests[j].Name, "Test")
		return nameI < nameJ
	})

	// Table Header
	headerIcon := tableHeaderStyle.Render(iconColStyle.Render("")) // Empty, but maintains width
	headerStatus := tableHeaderStyle.Render(statusColStyle.Render("RES"))
	headerName := tableHeaderStyle.Render(testNameColStyle.Render("TEST NAME"))
	headerDuration := tableHeaderStyle.Render(durationColStyle.Render("DURATION"))

	tableHeaderRow := lipgloss.JoinHorizontal(lipgloss.Left,
		headerIcon,
		strings.Repeat(" ", columnPadding),
		headerStatus,
		strings.Repeat(" ", columnPadding),
		headerName,
		strings.Repeat(" ", columnPadding),
		headerDuration,
	)
	fmt.Printf("%s\n", tableHeaderRow)
	fmt.Println(packageSeparatorStyle.Render(strings.Repeat("─", 80)))

	for _, testResult := range pkgResults.Tests {
		// Render and print each test row
		fmt.Println(displayTestTableRow(testResult, iconColStyle, statusColStyle, testNameColStyle, durationColStyle, columnPadding))

		// Show output for failed tests immediately below their table row
		if testResult.Status == StatusFail && len(testResult.Output) > 0 {
			for _, line := range testResult.Output {
				if strings.TrimSpace(line) != "" {
					fmt.Println(outputStyle.Render(line))
				}
			}
		}
	}
	fmt.Println() // Blank line after each package for separation
}

// displayTestTableRow prints the detailed outcome for an individual test as a table row.
func displayTestTableRow(
	result TestResult,
	iconColStyle, statusColStyle, testNameColStyle, durationColStyle lipgloss.Style,
	padding int,
) string {
	icon, statusFgStyle := getStatusInfo(result.Status)

	// Strip "Test" prefix here for display
	displayTestName := strings.TrimPrefix(result.Name, "Test")

	// Render each cell's content, then wrap it in the column's style for width/alignment
	iconCell := iconColStyle.Render(statusFgStyle.Render(icon))
	statusCell := statusColStyle.Render(statusFgStyle.Render(strings.ToUpper(result.Status)))
	nameCell := testNameColStyle.Render(testNameStyle.Render(displayTestName)) // Use stripped name
	durationCell := durationColStyle.Render(durationStyle.Render(fmt.Sprintf("(%v)", result.Duration)))

	// Join them horizontally with padding
	return lipgloss.JoinHorizontal(lipgloss.Left,
		iconCell,
		strings.Repeat(" ", padding),
		statusCell,
		strings.Repeat(" ", padding),
		nameCell,
		strings.Repeat(" ", padding),
		durationCell,
	)
}

// displayOverallSummary prints the final summary of all tests across all packages.
func displayOverallSummary(summary *TestSummary) {
	var summaryText strings.Builder

	summaryText.WriteString("Overall Test Summary\n\n")
	summaryText.WriteString(fmt.Sprintf("Total:   %d\n", summary.Total))
	summaryText.WriteString(fmt.Sprintf("%s %d\n", passStyle.Render("Passed:"), summary.Passed))
	summaryText.WriteString(fmt.Sprintf("%s %d\n", failStyle.Render("Failed:"), summary.Failed))
	summaryText.WriteString(fmt.Sprintf("%s %d", skipStyle.Render("Skipped:"), summary.Skipped))

	fmt.Println(summaryBoxStyle.Render(summaryText.String()))
}
