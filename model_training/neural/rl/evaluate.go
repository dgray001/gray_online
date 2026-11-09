package rl

import (
	"fiddlesticks.live/neural"
	"fiddlesticks.live/neural/experiment"
)

func evaluate(name string, network *neural.Network, scenarios []experiment.Scenario) (Evaluation, []Episode) {
	episodes := make([]Episode, len(scenarios))
	for i, scenario := range scenarios {
		episodes[i] = play(network, scenario, nil, nil)
	}
	return summarize(name, episodes), episodes
}
