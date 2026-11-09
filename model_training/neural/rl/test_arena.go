package rl

import (
	"fmt"
	"path/filepath"
)

func testArena(options Options, selected []Result, direct bool) ([]Evaluation, []Comparison, error) {
	names, models, err := testModels(selected, options)
	if err != nil {
		return nil, nil, err
	}
	kind, scenarios := "mixed", schedule(options.TestGames, 1601)
	if direct {
		kind, scenarios = "theory4", directSchedule(options.TestGames, 1602)
	}
	evaluations, comparisons, games := []Evaluation{}, []Comparison{}, map[string][]Episode{}
	for i, model := range models {
		evaluation, episodes := evaluate(names[i], model, scenarios)
		evaluations = append(evaluations, evaluation)
		games[names[i]] = episodes
		if direct && i >= 3 {
			comparisons = append(comparisons, pairedComparison(names[i], episodes, games["theory4"]))
		}
		fmt.Printf("%s %s exact=%.4f wins=%.4f margin=%+.4f\n", kind, names[i], evaluation.All.ExactRate, evaluation.All.WinShare, evaluation.All.Margin)
	}
	err = writeJSON(filepath.Join(options.Output, kind+"-games.json"), games)
	return evaluations, comparisons, err
}
