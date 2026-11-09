package rl

import (
	"fiddlesticks.live/neural"
	"fiddlesticks.live/neural/experiment"
)

func fullCases(players int) []experiment.Scenario {
	result := []experiment.Scenario{}
	lineups, names := [][]uint8{{0}, {2}, {4}, {0, 2, 4}}, []string{"uniform", "theory2", "theory4", "mixed"}
	for _, points := range [][2]uint16{{10, 1}, {5, 1}, {0, 5}, {10, 0}, {1, 5}} {
		for i, lineup := range lineups {
			result = append(result, experiment.Scenario{Settings: neural.GameSettings{Players: players, MaxRound: uint8(51 / players), RoundPoints: points[0], TrickPoints: points[1], OpponentModels: lineup}, Opponents: names[i]})
		}
	}
	return result
}
