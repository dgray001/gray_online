package rl

import (
	"fiddlesticks.live/neural/experiment"
	"math"
)

func trainingSchedule(count int, seed int64, coreShare float64) []experiment.Scenario {
	bag := []int{}
	for players := 2; players <= 10; players++ {
		weight := int(math.Round((1 - coreShare) * 50))
		if players >= 4 && players <= 8 {
			weight = int(math.Round(coreShare * 40))
		}
		for i := 0; i < weight; i++ {
			bag = append(bag, players)
		}
	}
	return buildSchedule(count, seed, bag)
}
