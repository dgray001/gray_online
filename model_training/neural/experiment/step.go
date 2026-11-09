package experiment

import "fiddlesticks.live/neural"

func trainEpoch(network *neural.Network, data []neural.Example, trial Trial) float64 {
	loss := 0.0
	for _, example := range data {
		if trial.Method == "soft" {
			loss += network.TrainDistribution(example.Input, example.Legal, example.Distribution, trial.Rate)
		} else {
			loss += network.Train(example.Input, example.Legal, example.Target, trial.Rate)
		}
	}
	return loss / float64(len(data))
}
