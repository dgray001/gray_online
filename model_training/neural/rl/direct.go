package rl

import "fiddlesticks.live/neural/experiment"

func directSchedule(count int, seed int64) []experiment.Scenario {
	result := schedule(count, seed)
	for i := range result {
		result[i].Opponents = "theory4"
		result[i].Settings.OpponentModels = []uint8{4}
	}
	return result
}
