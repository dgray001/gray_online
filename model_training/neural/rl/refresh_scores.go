package rl

import (
	"os"
	"path/filepath"
	"strings"
)

type savedSummary struct {
	Regimens    []Regimen
	Mixed       []Evaluation
	Theory4     []Evaluation
	Comparisons []Comparison
	Historical  []Evaluation
	Widths      []WidthResult
}

func RefreshScores(directory string) error {
	var manifest struct{ Options Options }
	var results []Result
	var summary savedSummary
	if err := readJSON(filepath.Join(directory, "manifest.json"), &manifest); err != nil {
		return err
	}
	if err := readJSON(filepath.Join(directory, "trials.json"), &results); err != nil {
		return err
	}
	if err := readJSON(filepath.Join(directory, "summary.json"), &summary); err != nil {
		return err
	}
	old, err := os.ReadFile(filepath.Join(directory, "report.md"))
	if err != nil {
		return err
	}
	suffix := ""
	if index := strings.Index(string(old), "\n## Findings\n"); index >= 0 {
		suffix = string(old[index:])
	}
	manifest.Options.Output = directory
	return refreshArenas(manifest.Options, results, summary, suffix)
}
