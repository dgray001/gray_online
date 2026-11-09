package rl

import (
	"fiddlesticks.live/neural"
	"fiddlesticks.live/neural/experiment"
	"path/filepath"
)

type fitting struct {
	network   *neural.Network
	best      *neural.Network
	result    Result
	learner   *Learner
	reference float64
	stale     int
}

func fit(trial Trial, options Options, scenarios []experiment.Scenario) (Result, error) {
	network := initialNetwork(trial)
	initial, _ := evaluate(trial.ID, network, scenarios)
	f := fitting{network: network, best: network.Clone(), result: Result{Trial: trial, Stop: "epoch cap", Initial: initial.All, InitialCore: selectionMetrics(initial), Validation: initial, Model: filepath.Join(options.Output, trial.ID+".json")}, learner: &Learner{Rate: trial.Rate, Entropy: trial.Entropy, Baselines: map[Context]Baseline{}}, reference: selectionMetrics(initial).ExactRate}
	if err := network.Save(f.result.Model); err != nil {
		return f.result, err
	}
	return f.run(options, scenarios)
}

func initialNetwork(trial Trial) *neural.Network {
	inputs := trial.Inputs
	if inputs == 0 {
		inputs = neural.LegacyInputs
	}
	network := neural.NewNetwork(inputs, trial.Hidden, neural.Outputs, trial.Seed)
	if inputs == neural.Inputs {
		network.FeatureVersion = neural.FeatureVersion
	} else {
		network.FeatureVersion = 1
	}
	network.BidPolicy = "categorical"
	if trial.Start == "zero" {
		for _, weights := range network.OutputWeights {
			clear(weights)
		}
	}
	if trial.Start == "small" {
		for _, matrix := range [][][]float64{network.HiddenWeights, network.OutputWeights} {
			for _, row := range matrix {
				for j := range row {
					row[j] *= 0.25
				}
			}
		}
	}
	return network
}
