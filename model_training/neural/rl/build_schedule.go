package rl

import (
	"fiddlesticks.live/neural/experiment"
	"math/rand"
	"slices"
)

func buildSchedule(count int, seed int64, bag []int) []experiment.Scenario {
	rng, players := rand.New(rand.NewSource(seed)), slices.Clone(bag)
	cases, used := map[int][]experiment.Scenario{}, map[int]int{}
	result := make([]experiment.Scenario, count)
	for i := range result {
		if i%len(players) == 0 {
			rng.Shuffle(len(players), func(a int, b int) { players[a], players[b] = players[b], players[a] })
		}
		p := players[i%len(players)]
		if cases[p] == nil {
			cases[p] = fullCases(p)
		}
		if used[p]%len(cases[p]) == 0 {
			rng.Shuffle(len(cases[p]), func(a int, b int) { cases[p][a], cases[p][b] = cases[p][b], cases[p][a] })
		}
		result[i] = cases[p][used[p]%len(cases[p])]
		result[i].Seat = used[p] % p
		used[p]++
		result[i].Settings.DealSeed, result[i].ActionSeed = seed*1000000+int64(i)+1, seed*1000000+int64(i)+500001
	}
	return result
}
