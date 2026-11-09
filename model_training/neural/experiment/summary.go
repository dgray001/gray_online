package experiment

import (
	"fmt"
	"path/filepath"
)

func saveSummary(options Options, results []Result, ranking []Regimen, validation []Evaluation, test []Evaluation, tournament []Evaluation) error {
	value := struct {
		Regimens   []Regimen
		Validation []Evaluation
		Test       []Evaluation
		Tournament []Evaluation
	}{ranking, validation, test, tournament}
	if err := writeJSON(filepath.Join(options.Output, "summary.json"), value); err != nil {
		return err
	}
	if err := report(options, results, ranking, test, tournament); err != nil {
		return err
	}
	fmt.Printf("report: %s\n", filepath.Join(options.Output, "report.md"))
	return nil
}
