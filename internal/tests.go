package internal

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/lipgloss/v2/tree"
)

// --- Constants for Test Statuses ---
const (
	StatusRun     Status = "run"
	StatusPass    Status = "pass"
	StatusFail    Status = "fail"
	StatusSkip    Status = "skip"
	StatusOutput  Status = "output"
	StatusGroup   Status = "group"   // Used for test/benchmark groups
	StatusRunning Status = "running" // Internal status for tests currently executing
)

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
	Name         string
	Tests        []TestResult
	Status       Status // Derived: StatusPass, StatusFail, StatusSkip
	Total        int
	Passed       int
	Failed       int
	Skipped      int
	Duration     time.Duration // Sum of individual test durations in the package
	Coverage     string        // Raw `coverage:` line from go test, empty if -cover wasn't used
	FuncCoverage []FuncCoverage
}

// --- TestSummary (Overall results of the entire test run) ---
type TestSummary struct {
	sync.Mutex              // Protects global counters
	Results    []TestResult // Flat list of all individual test results
	Passed     int
	Failed     int
	Skipped    int
	Total      int

	PackageCoverage     map[string]string // Per package coverage line from go test -cover
	PackageFuncCoverage map[string][]FuncCoverage
	TotalCoverage       string
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
				Name:     pkgName,
				Tests:    []TestResult{},
				Status:   StatusPass,
				Coverage: overallSummary.PackageCoverage[pkgName],
			}
			if funcs, ok := overallSummary.PackageFuncCoverage[pkgName]; ok {
				groupedByPackage[pkgName].FuncCoverage = funcs
			} else if slash := strings.LastIndex(pkgName, "/"); slash >= 0 {
				if funcs, ok := overallSummary.PackageFuncCoverage[pkgName[slash+1:]]; ok {
					groupedByPackage[pkgName].FuncCoverage = funcs
				}
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
		}
	}

	packageNames := make([]string, 0, len(groupedByPackage))
	for pkgName := range groupedByPackage {
		packageNames = append(packageNames, pkgName)
	}
	sort.Strings(packageNames)

	if !GlobalConfig.SummaryOnly {
		for _, pkgName := range packageNames {
			pkgResults := groupedByPackage[pkgName]
			renderBlocks = append(renderBlocks, displayPackageBlock(pkgResults))
		}
	}

	// Overall summary
	if GlobalConfig.SummaryOnly || len(groupedByPackage) > 1 {
		renderBlocks = append(renderBlocks, displayOverallSummary(overallSummary))
	}

	mainChunk := lipgloss.JoinVertical(lipgloss.Left, renderBlocks...)

	// Join all blocks with two newlines for separation (a blank line between them)
	lipgloss.Println(AppOverallOutputStyle.Render(mainChunk))
}

// displayPackageBlock builds and returns the display string for a single package.
// It returns a string without a trailing newline.
func displayPackageBlock(pkgResults *PackageResults) string {
	if pkgResults.Total == pkgResults.Skipped {
		pkgResults.Status = StatusSkip
	}

	pkgHeader := fmt.Sprintf("%v %v %v", pkgResults.Status.String(), packageStyle.Render(pkgResults.Name), durationStyle.Render(fmt.Sprintf("(%v)", pkgResults.Duration)))

	pkgTestResults := fmt.Sprintf(
		"%d total • %s • %s • %s",
		pkgResults.Total,
		passStyle.Render(fmt.Sprintf("%d passed", pkgResults.Passed)),
		failStyle.Render(fmt.Sprintf("%d failed", pkgResults.Failed)),
		skipStyle.Render(fmt.Sprintf("%d skipped", pkgResults.Skipped)),
	)

	if cov := parseCoverage(pkgResults.Coverage); cov != "" {
		pkgTestResults += " • " + durationStyle.Render(cov+" covered")
	}

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

	testBlock := generateTestBlock(pkgResults.Tests)

	// Build package-level coverage block (tree-style with lipgloss tree)
	var covBlock []string
	if len(pkgResults.FuncCoverage) > 0 {
		covHeader := "Coverage Summary"
		if covPct := parseCoverage(pkgResults.Coverage); covPct != "" {
			covHeader += fmt.Sprintf(" (%s covered)", packageStyle.Render(covPct))
		}
		covLines := formatFuncCoverage(pkgResults.FuncCoverage, covHeader)
		for line := range strings.SplitSeq(covLines, "\n") {
			covBlock = append(covBlock, line)
		}
	}

	blockWidth := lipgloss.Width(pkgHeader)
	for _, l := range testBlock {
		if w := lipgloss.Width(l); w > blockWidth {
			blockWidth = w
		}
	}
	for _, l := range covBlock {
		if w := lipgloss.Width(l); w > blockWidth {
			blockWidth = w
		}
	}
	separatorLine := packageSeparatorStyle.Render(strings.Repeat("─", blockWidth))

	items := make([]string, 0, 3+len(testBlock)+len(covBlock)+2)
	items = append(items, pkgHeader, pkgTestResults, separatorLine)
	items = append(items, testBlock...)
	items = append(items, covBlock...)

	items = append(items, "")

	return lipgloss.JoinVertical(lipgloss.Left, items...)
}

// generateTestBlock builds the display lines for tests.
func generateTestBlock(tests []TestResult) []string {
	var lines []string
	for _, result := range tests {
		if GlobalConfig.OnlyFails && !(result.Status == StatusFail) {
			continue
		}

		displayTestName := strings.TrimPrefix(result.Name, "Test")

		// Test header:  ✗ FAIL  5.01s  FailingTest
		line := fmt.Sprintf("%s  %s  %s",
			result.Status.String(),
			durationStyle.Render(fmt.Sprintf("%v", result.Duration)),
			testNameStyle.Render(displayTestName),
		)
		lines = append(lines, line)

		if len(result.Output) > 0 && GlobalConfig.Verbose {
			for _, ol := range result.Output {
				if strings.TrimSpace(ol) != "" && !(strings.HasPrefix(ol, "===") || strings.HasPrefix(ol, "---")) {
					lines = append(lines, "   "+outputStyle.Render(ol))
				}
			}
		}
	}
	return lines
}

// coverageStyleForPct returns a color style based on coverage percentage threshold.
func coverageStyleForPct(pct float64) lipgloss.Style {
	switch {
	case pct >= 80:
		return passStyle
	case pct >= 50:
		return skipStyle
	default:
		return failStyle
	}
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
	if summary.TotalCoverage != "" {
		out += " • " + durationStyle.Render(summary.TotalCoverage+"% covered")
	}
	if !GlobalConfig.NoBar {
		out += "\n" + renderProportionalBar(summary, lipgloss.Width(out))
	}
	return pkgTableStyle.AlignHorizontal(lipgloss.Left).MarginBottom(0).Render(out)
}

func renderProportionalBar(summary *TestSummary, width int) string {
	passWidth := int(float64(summary.Passed) / float64(summary.Total) * float64(width))
	failWidth := int(float64(summary.Failed) / float64(summary.Total) * float64(width))
	remainder := width - passWidth - failWidth
	skipWidth := 0

	if summary.Skipped == 0 {
		failWidth += remainder
	} else {
		skipWidth = remainder
	}

	passBar := passStyle.Render(strings.Repeat("━", passWidth))
	failBar := failStyle.Render(strings.Repeat("━", failWidth))
	skipBar := skipStyle.Render(strings.Repeat("━", skipWidth))

	return lipgloss.JoinHorizontal(lipgloss.Top, passBar, failBar, skipBar)
}

func parseCoverage(raw string) string {
	if !strings.HasPrefix(raw, "coverage: ") {
		return ""
	}
	rest := strings.TrimPrefix(raw, "coverage: ")
	if rest == "[no statements]" {
		return ""
	}
	if stripped, ok := strings.CutSuffix(rest, " of statements"); ok {
		return stripped
	}
	return rest
}

// formatFuncCoverage renders per-function coverage as a lipgloss tree.
// The header is the tree root; each function is a child with name and | XX%.
func formatFuncCoverage(funcs []FuncCoverage, header string) string {
	if len(funcs) == 0 {
		return ""
	}

	// Find max display width of styled function names for | alignment
	maxNameWidth := 0
	nameStrings := make([]string, len(funcs))
	for i, fc := range funcs {
		nameStrings[i] = testNameStyle.Render(fc.Name)
		if w := lipgloss.Width(nameStrings[i]); w > maxNameWidth {
			maxNameWidth = w
		}
	}

	items := make([]any, len(funcs))
	for i, fc := range funcs {
		padded := nameStrings[i] + strings.Repeat(" ", maxNameWidth-lipgloss.Width(nameStrings[i]))
		pctStr := coverageStyleForPct(fc.Pct).Render(fmt.Sprintf("| %3.0f%%", fc.Pct))
		items[i] = fmt.Sprintf("%s %s", padded, pctStr)
	}

	t := tree.New().
		Root(header).
		EnumeratorStyle(coveragePipeStyle).
		Child(items...)

	return t.String()
}
