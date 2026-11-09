package behavior

import (
	"fiddlesticks.live/neural"
	"math"
)

func predict(network *neural.Network, example neural.Example) Prediction {
	input := example.Input
	if network.InputCount() == neural.LegacyInputs && len(input) == neural.Inputs {
		input = input[:neural.LegacyInputs]
	}
	probabilities := network.Predict(input, example.Legal)
	result := Prediction{Probabilities: make([]float64, len(example.Legal)), Choice: example.Legal[0]}
	for i, action := range example.Legal {
		result.Probabilities[i] = probabilities[action]
		if probabilities[action] > probabilities[result.Choice] {
			result.Choice = action
		}
		if example.Legal[0] < 27 {
			result.MeanBid += float64(action) * probabilities[action]
		}
	}
	if example.Legal[0] < 27 && network.BidPolicy != "categorical" {
		result.MeanBid = math.Min(float64(len(example.Legal)-1), result.MeanBid)
		result.Probabilities = bidPolicy(result.MeanBid, len(example.Legal))
		result.Choice = int(math.Round(result.MeanBid))
	}
	return result
}
