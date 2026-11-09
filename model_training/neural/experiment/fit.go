package experiment

import (
	"fiddlesticks.live/neural"
	"math/rand"
	"slices"
)

func fit(trial Trial, training []neural.Example, validation []neural.Example, options Options) (Result, *neural.Network) {
	network := neural.NewNetwork(neural.Inputs, 0, neural.Outputs, trial.Seed)
	if trial.Start == "zero" {
		for _, row := range network.OutputWeights {
			clear(row)
		}
	}
	best := network.Clone()
	bestLoss := measure(network, validation, trial.Method).Loss
	reference, stale := bestLoss, 0
	result := Result{Trial: trial, Stop: "epoch cap"}
	data, rng := slices.Clone(training), rand.New(rand.NewSource(trial.Seed))
	return fitEpochs(network, best, result, data, validation, options, rng, bestLoss, reference, stale)
}
