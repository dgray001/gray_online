package rl

import (
	"fiddlesticks.live/neural"
	"fmt"
)

func trials(options Options) []Trial {
	if len(options.Widths) > 0 {
		return widthSweep(options)
	}
	if !options.Sweep {
		return []Trial{configuredTrial(options.Hidden, options.Start, options.Rate, options.Entropy, options.Seed)}
	}
	result := []Trial{}
	starts := []string{"zero", "xavier"}
	if options.Hidden > 0 {
		starts = []string{"xavier", "small"}
	}
	for _, start := range starts {
		for _, rate := range []float64{0.0005, 0.002} {
			for _, entropy := range []float64{0, 0.01} {
				for _, seed := range []int64{1, 2, 3} {
					result = append(result, configuredTrial(options.Hidden, start, rate, entropy, seed))
				}
			}
		}
	}
	if options.LinearControls && options.Hidden > 0 {
		for _, seed := range []int64{1, 2, 3} {
			result = append(result, configuredTrial(0, "xavier", 0.002, 0, seed))
		}
	}
	return result
}

func trial(start string, rate float64, entropy float64, seed int64) Trial {
	return Trial{Inputs: neural.Inputs, ID: fmt.Sprintf("rl-%s-r%.4f-e%.2f-s%d", start, rate, entropy, seed), Seed: seed, Rate: rate, Entropy: entropy, Start: start}
}
