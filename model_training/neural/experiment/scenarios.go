package experiment

import (
	"fiddlesticks.live/neural"
	"math/rand"
)

type Scenario struct {
	Settings   neural.GameSettings
	Seat       int
	Opponents  string
	ActionSeed int64
}

func cases() []Scenario {
	result := []Scenario{}
	lineups := [][]uint8{{0}, {2}, {4}, {0, 2, 4}}
	names := []string{"random", "theory2", "theory4", "mixed"}
	for _, players := range []int{2, 3, 4, 5, 6} {
		for _, rounds := range []uint8{2, 4, 6, 8} {
			for _, scoring := range [][2]uint16{{10, 1}, {5, 1}, {0, 5}, {10, 0}, {1, 5}} {
				for i, lineup := range lineups {
					result = append(result, Scenario{Settings: neural.GameSettings{Players: players, MaxRound: rounds, RoundPoints: scoring[0], TrickPoints: scoring[1], OpponentModels: lineup}, Opponents: names[i]})
				}
			}
		}
	}
	return result
}

func scenarios(count int, seed int64) []Scenario {
	base := cases()
	rng := rand.New(rand.NewSource(seed))
	rng.Shuffle(len(base), func(i int, j int) { base[i], base[j] = base[j], base[i] })
	result := make([]Scenario, count)
	for i := range result {
		result[i] = base[i%len(base)]
		result[i].Seat = (i/len(base) + i) % result[i].Settings.Players
		result[i].Settings.DealSeed = seed*1000000 + int64(i) + 1
		result[i].ActionSeed = seed*1000000 + int64(i) + 500001
	}
	return result
}
