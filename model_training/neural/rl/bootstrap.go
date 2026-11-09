package rl

import (
	"math/rand"
	"slices"
)

type Comparison struct {
	Agent       string
	ExactDelta  float64
	Low         float64
	High        float64
	WinDelta    float64
	MarginDelta float64
}

func pairedComparison(name string, agent []Episode, baseline []Episode) Comparison {
	a, b := summarize(name, agent).All, summarize("theory4", baseline).All
	differences := make([]float64, 2000)
	rng := rand.New(rand.NewSource(1))
	for sample := range differences {
		delta, rounds := 0, 0
		for range agent {
			i := rng.Intn(len(agent))
			rounds += len(agent[i].Outcomes)
			delta += exactCount(agent[i]) - exactCount(baseline[i])
		}
		differences[sample] = float64(delta) / float64(rounds)
	}
	slices.Sort(differences)
	return Comparison{Agent: name, ExactDelta: a.ExactRate - b.ExactRate, Low: differences[50], High: differences[1949], WinDelta: a.WinShare - b.WinShare, MarginDelta: a.Margin - b.Margin}
}
