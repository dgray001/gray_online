package main

import (
	"fiddlesticks.live/neural"
	"fmt"
	"slices"
)

func benchmark(network *neural.Network, settings neural.GameSettings, games int) {
	wins, ties, total, opponents := 0, 0, 0.0, 0.0
	for i := 0; i < games; i++ {
		seat := i % settings.Players
		scores := neural.Play(network, nil, settings, seat)
		maximum, leaders := slices.Max(scores), 0
		for j, score := range scores {
			if score == maximum {
				leaders++
			}
			if j != seat {
				opponents += float64(score)
			}
		}
		if scores[seat] == maximum {
			if leaders == 1 {
				wins++
			} else {
				ties++
			}
		}
		total += float64(scores[seat])
	}
	fmt.Printf("imitation evaluation %d games: wins %d ties %d mean score %.2f opponent mean %.2f\n", games, wins, ties, total/float64(games), opponents/float64(games*(settings.Players-1)))
}
