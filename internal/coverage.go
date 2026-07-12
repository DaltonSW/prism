package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
