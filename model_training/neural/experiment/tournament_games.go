package experiment

import (
	"fiddlesticks.live/neural"
	"math/rand"
)

func tournamentGames(models []*neural.Network, names []string, games int) []GameResult {
	rows := []GameResult{}
	scoring, rounds := [][2]uint16{{10, 1}, {5, 1}, {0, 5}, {10, 0}, {1, 5}}, []uint8{2, 4, 6, 8}
	for i := 0; i < games; i++ {
		cycle := i / len(models)
		settings := neural.GameSettings{Players: len(models), MaxRound: rounds[cycle/5%4], RoundPoints: scoring[cycle%5][0], TrickPoints: scoring[cycle%5][1], OpponentModels: []uint8{4}, DealSeed: 70000000 + int64(cycle), Networks: make([]*neural.Network, len(models))}
		for seat := range models {
			settings.Networks[seat] = models[(seat+i)%len(models)]
		}
		rand.Seed(71000000 + int64(cycle))
		scores := neural.Play(nil, nil, settings, 0)
		for seat := range models {
			rows = append(rows, scoredGame(names[(seat+i)%len(models)], Scenario{Settings: settings, Seat: seat, Opponents: "tournament", ActionSeed: 71000000 + int64(cycle)}, scores))
		}
	}
	return rows
}
