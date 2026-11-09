package rl

import (
	"fiddlesticks.live/neural/experiment"
	"fmt"
	"os"
	"path/filepath"
)

func CompareModels(directory string) error {
	var manifest struct {
		Options   Options
		MixedTest []experiment.Scenario
	}
	var summary savedSummary
	if err := readJSON(filepath.Join(directory, "manifest.json"), &manifest); err != nil {
		return err
	}
	if err := readJSON(filepath.Join(directory, "summary.json"), &summary); err != nil {
		return err
	}
	agents, err := comparisonAgents(directory, manifest.Options, summary)
	if err != nil {
		return err
	}
	output := filepath.Join(directory, "model-comparison")
	if err := os.MkdirAll(output, 0755); err != nil {
		return err
	}
	results, games := compareAgents(agents, manifest.MixedTest)
	if err := writeJSON(filepath.Join(output, "summary.json"), results); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(output, "games.json"), games); err != nil {
		return err
	}
	if err := comparisonReport(output, results); err != nil {
		return err
	}
	fmt.Printf("model comparison: %s\n", filepath.Join(output, "report.md"))
	return nil
}
