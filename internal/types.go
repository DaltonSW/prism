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
	StatusRun     Status = "run"
	StatusPass    Status = "pass"
	StatusFail    Status = "fail"
	StatusSkip    Status = "skip"
	StatusOutput  Status = "output"
	StatusRunning Status = "running" // Internal status for tests currently executing
)

var GlobalConfig Config

type Config struct {
	Verbose bool
}

type Status string

func (s Status) String() string {
	var icon string
	var style lipgloss.Style
	switch s {
	case StatusPass:
		icon, style = "✓", passStyle
	case StatusFail:
		icon, style = "✗", failStyle
	case StatusSkip:
		icon, style = "⊝", skipStyle
	default:
		icon, style = "◌", lipgloss.NewStyle().Foreground(lipgloss.Color("#B0B0B0"))
	}

	return style.Render(fmt.Sprintf("%v %v", icon, strings.ToUpper(string(s))))
}

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
	Status   Status // StatusPass, StatusFail, StatusSkip, StatusRunning
	Duration time.Duration
	Output   []string // Raw output from the test
}

// --- PackageResults (Aggregated results for a single package) ---
type PackageResults struct {
	Name     string
	Tests    []TestResult
	Status   Status // Derived: StatusPass, StatusFail, StatusSkip
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
func displayResults(overallSummary *TestSummary) {
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
	if len(groupedByPackage) > 1 {
		renderBlocks = append(renderBlocks, displayOverallSummary(overallSummary))
	}

	// Join all blocks with two newlines for separation (a blank line between them)
	fmt.Println(AppOverallOutputStyle.Render(lipgloss.JoinVertical(lipgloss.Left, renderBlocks...)))
}

// displayPackageBlock builds and returns the display string for a single package.
// It returns a string without a trailing newline.
func displayPackageBlock(pkgResults *PackageResults) string {
	pkgHeader := lipgloss.JoinHorizontal(
		lipgloss.Left,
		pkgResults.Status.String(),
		" ",
		packageStyle.Render(pkgResults.Name),
		" ",
		durationStyle.Render(fmt.Sprintf("(%v)", pkgResults.Duration)),
	)

	pkgHeader = lipgloss.JoinVertical(lipgloss.Left, pkgHeader,
		fmt.Sprintf(
			"%d total • %s • %s • %s",
			pkgResults.Total,
			passStyle.Render(fmt.Sprintf("%d passed", pkgResults.Passed)),
			failStyle.Render(fmt.Sprintf("%d failed", pkgResults.Failed)),
			skipStyle.Render(fmt.Sprintf("%d skipped", pkgResults.Skipped)),
		),
	)

	sort.Slice(pkgResults.Tests, func(i, j int) bool {
		statusOrder := map[Status]int{
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
		Headers("RESULT", "DUR", "TEST").
		Rows(generateTestRows(pkgResults.Tests)...)

	tableStr := t.Render()

	separatorLine := packageSeparatorStyle.Render(strings.Repeat("─", max(lipgloss.Width(tableStr), lipgloss.Width(pkgHeader))))

	return lipgloss.JoinVertical(lipgloss.Left,
		pkgHeader,
		separatorLine,
		pkgTableStyle.Render(tableStr),
	)
}

// generateTestRows creates the rows for the lipgloss table.
// This helper function remains, returning [][]string data.
func generateTestRows(tests []TestResult) [][]string {
	rows := make([][]string, 0) // Initialize with 0 capacity as output lines are dynamic
	for _, result := range tests {
		displayTestName := strings.TrimPrefix(result.Name, "Test")

		row := []string{
			result.Status.String(),
			durationStyle.Render(fmt.Sprintf("%v", result.Duration)),
			testNameStyle.Render(displayTestName),
		}
		rows = append(rows, row)

		if result.Status == StatusFail && len(result.Output) > 0 && GlobalConfig.Verbose {
			for _, line := range result.Output {
				if strings.TrimSpace(line) != "" && !(strings.HasPrefix(line, "===") || strings.HasPrefix(line, "---")) {
					outputRow := []string{"", "", outputStyle.Render(line)}
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
	out += fmt.Sprintf(
		"%d total • %s • %s • %s",
		summary.Total,
		passStyle.Render(fmt.Sprintf("%d passed", summary.Passed)),
		failStyle.Render(fmt.Sprintf("%d failed", summary.Failed)),
		skipStyle.Render(fmt.Sprintf("%d skipped", summary.Skipped)),
	)

	return pkgTableStyle.Padding(1).Render(out)
}
