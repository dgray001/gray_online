package rl

import "fiddlesticks.live/neural/experiment"

func schedule(count int, seed int64) []experiment.Scenario {
	return buildSchedule(count, seed, []int{2, 3, 4, 5, 6, 7, 8, 9, 10})
}
