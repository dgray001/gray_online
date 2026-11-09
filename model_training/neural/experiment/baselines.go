package experiment

import (
	"fiddlesticks.live/neural"
	"fmt"
)

func baselines(schedule []Scenario) ([]GameResult, []Evaluation) {
	rows, summaries := arena("theory4", nil, schedule)
	for seed := int64(0); seed <= 3; seed++ {
		network := neural.NewNetwork(neural.Inputs, 0, neural.Outputs, max(1, seed))
		name := fmt.Sprintf("untrained-xavier-%d", seed)
		if seed == 0 {
			name = "uniform"
			for _, weights := range network.OutputWeights {
				clear(weights)
			}
		}
		games, summary := arena(name, network, schedule)
		rows, summaries = append(rows, games...), append(summaries, summary...)
	}
	return rows, summaries
}
