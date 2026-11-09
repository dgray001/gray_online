package rl

import (
	"fiddlesticks.live/neural/experiment"
	"path/filepath"
)

func historicalArena(options Options, selected []Result) ([]Evaluation, error) {
	names, models, err := testModels(selected, options)
	if err != nil {
		return nil, err
	}
	cases := experiment.Scenarios(800, 601)
	for i := range cases {
		if cases[i].Opponents == "random" {
			cases[i].Opponents = "uniform"
		}
	}
	evaluations, games := []Evaluation{}, map[string][]Episode{}
	for i, model := range models {
		evaluation, episodes := evaluate(names[i], model, cases)
		evaluations = append(evaluations, evaluation)
		games[names[i]] = episodes
	}
	err = writeJSON(filepath.Join(options.Output, "historical-games.json"), games)
	return evaluations, err
}
