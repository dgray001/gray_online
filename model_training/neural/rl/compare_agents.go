package rl

import (
	"fiddlesticks.live/neural/experiment"
	"fmt"
)

func compareAgents(agents []benchmarkAgent, scenarios []experiment.Scenario) ([]benchmarkResult, map[string][]Episode) {
	results, games := []benchmarkResult{}, map[string][]Episode{}
	for _, candidate := range agents {
		audit := PolicyAudit{}
		episodes := make([]Episode, len(scenarios))
		for i, scenario := range scenarios {
			episodes[i] = comparisonGame(candidate, scenario, &audit)
		}
		evaluation := summarize(candidate.Name, episodes)
		results = append(results, benchmarkResult{Evaluation: evaluation, Policy: audit.averaged()})
		games[candidate.Name] = episodes
		core := selectionMetrics(evaluation)
		fmt.Printf("%s exact=%.3f points=%.2f wins=%.3f\n", candidate.Name, core.ExactRate, core.Points, core.WinShare)
	}
	return results, games
}
