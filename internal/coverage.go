package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
)

type FuncCoverage struct {
	Name string
	Pct  float64
}

func loadCoverage(profilePath string) (map[string][]FuncCoverage, string, error) {
	if profilePath == "" {
		return nil, "", nil
	}
	defer os.Remove(profilePath)

	cmd := exec.Command("go", "tool", "cover", "-func="+profilePath)
	out, err := cmd.Output()
	if err != nil {
		return nil, "", fmt.Errorf("go tool cover -func: %w", err)
	}

	return parseFuncCoverage(string(out))
}

func parseFuncCoverage(output string) (map[string][]FuncCoverage, string, error) {
	result := make(map[string][]FuncCoverage)
	var totalPct string

	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		if fields[0] == "total:" {
			totalPct = strings.TrimSuffix(fields[len(fields)-1], "%")
			continue
		}

		path, _, _ := strings.Cut(fields[0], ":")
		pctStr := strings.TrimSuffix(fields[len(fields)-1], "%")
		pct, err := strconv.ParseFloat(pctStr, 64)
		if err != nil {
			continue
		}

		pkg := filepath.Dir(path)
		if pkg == "." {
			pkg = ""
		}

		result[pkg] = append(result[pkg], FuncCoverage{
			Name: fields[1],
			Pct:  pct,
		})
	}

	return result, totalPct, nil
}

func formatFuncCoverage(funcs []FuncCoverage, width int) string {
	if len(funcs) == 0 {
		return ""
	}

	sorted := make([]FuncCoverage, len(funcs))
	copy(sorted, funcs)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Pct != sorted[j].Pct {
			return sorted[i].Pct < sorted[j].Pct
		}
		return sorted[i].Name < sorted[j].Name
	})

	rendered := make([]string, len(sorted))
	for i, fc := range sorted {
		var style lipgloss.Style
		switch {
		case fc.Pct >= 80:
			style = passStyle
		case fc.Pct >= 50:
			style = skipStyle
		default:
			style = failStyle
		}
		rendered[i] = style.Render(fmt.Sprintf("%s %.0f%%", fc.Name, fc.Pct))
	}

	const sep = "  "
	sepW := lipgloss.Width(sep)

	var lines []string
	var current []string
	currentWidth := 0
	for _, r := range rendered {
		w := lipgloss.Width(r)
		extra := w
		if len(current) > 0 {
			extra += sepW
		}
		if len(current) > 0 && currentWidth+extra > width {
			lines = append(lines, strings.Join(current, sep))
			current = []string{r}
			currentWidth = w
		} else {
			current = append(current, r)
			currentWidth += extra
		}
	}
	if len(current) > 0 {
		lines = append(lines, strings.Join(current, sep))
	}

	return strings.Join(lines, "\n")
}
