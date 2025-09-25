package internal

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/lipgloss/v2/table"
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

func (summary *TestSummary) String() string {
	return ""
}

// displayResults collects all rendered strings and returns them as a single output string.
func displayResults(overallSummary *TestSummary) string {
	var renderBlocks []string

	groupedByPackage := make(map[string]*PackageResults)
	for _, testResult := range overallSummary.Results {
		pkgName := testResult.Package
		if _, ok := groupedByPackage[pkgName]; !ok {
			groupedByPackage[pkgName] = &PackageResults{
				Name:   pkgName,
				Tests:  []TestResult{},
				Status: StatusPass,
			}
		}
		pkgResults := groupedByPackage[pkgName]
		pkgResults.Tests = append(pkgResults.Tests, testResult)
		pkgResults.Total++
		pkgResults.Duration += testResult.Duration

		switch testResult.Status {
		case StatusPass:
			pkgResults.Passed++
		case StatusFail:
			pkgResults.Failed++
			pkgResults.Status = StatusFail
		case StatusSkip:
			pkgResults.Skipped++
			if pkgResults.Status == StatusPass && pkgResults.Skipped == pkgResults.Total {
				pkgResults.Status = StatusSkip
			}
		}
	}

	packageNames := make([]string, 0, len(groupedByPackage))
	for pkgName := range groupedByPackage {
		packageNames = append(packageNames, pkgName)
	}
	sort.Strings(packageNames)

	for _, pkgName := range packageNames {
		pkgResults := groupedByPackage[pkgName]
		renderBlocks = append(renderBlocks, displayPackageBlock(pkgResults))
	}

	// Overall summary
	renderBlocks = append(renderBlocks, displayOverallSummary(overallSummary))

	// Join all blocks with two newlines for separation (a blank line between them)
	return AppOverallOutputStyle.Render(lipgloss.JoinVertical(lipgloss.Left, renderBlocks...))
}

// displayPackageBlock builds and returns the display string for a single package.
// It returns a string without a trailing newline.
func displayPackageBlock(pkgResults *PackageResults) string {
	icon, statusStyle := getStatusInfo(pkgResults.Status)
	statusText := strings.ToUpper(pkgResults.Status)

	pkgHeader := lipgloss.JoinHorizontal(
		lipgloss.Left,
		statusStyle.Render(icon),
		" ",
		statusStyle.Render(statusText),
		" ",
		packageStyle.Render(pkgResults.Name),
		" ",
		durationStyle.Render(fmt.Sprintf("(%v)", pkgResults.Duration)),
		lipgloss.NewStyle().Faint(true).Render(
			fmt.Sprintf(
				" (%d total, %s %d passed, %s %d failed, %s %d skipped)",
				pkgResults.Total,
				passStyle.Render(""), pkgResults.Passed,
				failStyle.Render(""), pkgResults.Failed,
				skipStyle.Render(""), pkgResults.Skipped,
			),
		),
	)

	separatorLine := packageSeparatorStyle.Render(strings.Repeat("─", lipgloss.Width(pkgHeader)))

	sort.Slice(pkgResults.Tests, func(i, j int) bool {
		statusOrder := map[string]int{
			StatusFail:    3,
			StatusSkip:    2,
			StatusPass:    1,
			StatusRunning: 0,
		}
		orderI := statusOrder[pkgResults.Tests[i].Status]
		orderJ := statusOrder[pkgResults.Tests[j].Status]

		if orderI != orderJ {
			return orderI < orderJ
		}
		nameI := strings.TrimPrefix(pkgResults.Tests[i].Name, "Test")
		nameJ := strings.TrimPrefix(pkgResults.Tests[j].Name, "Test")
		return nameI < nameJ
	})

	t := table.New().
		Border(lipgloss.HiddenBorder()).
		Width(lipgloss.Width(pkgHeader)).
		Headers(" ", "RES", "TEST NAME", "DURATION").
		Rows(generateTestRows(pkgResults.Tests)...)

	return lipgloss.JoinVertical(lipgloss.Left,
		pkgHeader,
		separatorLine,
		pkgTableStyle.Render(t.Render()),
		"\n\n",
	)
}

// generateTestRows creates the rows for the lipgloss table.
// This helper function remains, returning [][]string data.
func generateTestRows(tests []TestResult) [][]string {
	rows := make([][]string, 0) // Initialize with 0 capacity as output lines are dynamic
	for _, result := range tests {
		icon, statusFgStyle := getStatusInfo(result.Status)
		displayTestName := strings.TrimPrefix(result.Name, "Test")

		row := []string{
			statusFgStyle.Render(icon),
			statusFgStyle.Render(strings.ToUpper(result.Status)),
			testNameStyle.Render(displayTestName),
			durationStyle.Render(fmt.Sprintf("(%v)", result.Duration)),
		}
		rows = append(rows, row)

		if result.Status == StatusFail && len(result.Output) > 0 {
			for _, line := range result.Output {
				if strings.TrimSpace(line) != "" {
					outputRow := []string{"", "", outputStyle.Render(line), ""}
					rows = append(rows, outputRow)
				}
			}
		}
	}
	return rows
}

// displayOverallSummary builds and returns the display string for the overall summary.
func displayOverallSummary(summary *TestSummary) string {
	out := "Overall Test Results\n"
	out += fmt.Sprintf("Total: %d | Passed: %d | Skipped: %d | Failed: %d", summary.Total, summary.Passed, summary.Skipped, summary.Failed)

	return out
}

func getStatusInfo(status string) (icon string, style lipgloss.Style) {
	switch status {
	case StatusPass:
		return "✓", passStyle
	case StatusFail:
		return "✗", failStyle
	case StatusSkip:
		return "⊝", skipStyle
	default:
		return "◌", lipgloss.NewStyle().Foreground(lipgloss.Color("#B0B0B0"))
	}
}
