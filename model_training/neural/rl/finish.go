package rl

import (
	"fmt"
	"path/filepath"
)

func finish(options Options, results []Result) error {
	ranking := regimens(results)
	selected := champions(results, ranking)
	mixed, _, err := testArena(options, selected, false)
	if err != nil {
		return err
	}
	direct, comparisons, err := testArena(options, selected, true)
	if err != nil {
		return err
	}
	historical, err := historicalArena(options, selected)
	if err != nil {
		return err
	}
	value := struct {
		Regimens    []Regimen
		Mixed       []Evaluation
		Theory4     []Evaluation
		Comparisons []Comparison
		Historical  []Evaluation
		Widths      []WidthResult
	}{ranking, mixed, direct, comparisons, historical, widthResults(ranking, mixed, historical)}
	if err := writeJSON(filepath.Join(options.Output, "summary.json"), value); err != nil {
		return err
	}
	if err := report(options, results, ranking, mixed, direct, comparisons, historical); err != nil {
		return err
	}
	fmt.Printf("RL report: %s\n", filepath.Join(options.Output, "report.md"))
	return nil
}
