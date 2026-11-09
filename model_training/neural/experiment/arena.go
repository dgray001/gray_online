package experiment

import (
	"fiddlesticks.live/neural"
	"math/rand"
)

func arena(name string, network *neural.Network, schedule []Scenario) ([]GameResult, []Evaluation) {
	rows := make([]GameResult, 0, len(schedule))
	for _, scenario := range schedule {
		rand.Seed(scenario.ActionSeed)
		scores := neural.Play(network, nil, scenario.Settings, scenario.Seat)
		rows = append(rows, scoredGame(name, scenario, scores))
	}
	return rows, summarize(rows)
}
