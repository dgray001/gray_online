package rl

import "fiddlesticks.live/neural"

func (a *Agent) choose(input []float64, legal []int) int {
	probabilities := make([]float64, neural.Outputs)
	var hidden []float64
	if a.Network == nil || (a.RandomCards && legal[0] >= 27) {
		for _, action := range legal {
			probabilities[action] = 1 / float64(len(legal))
		}
	} else {
		probabilities, hidden = a.Network.PredictWithHidden(input, legal)
	}
	if a.ZeroBid && legal[0] < 27 {
		clear(probabilities)
		probabilities[0] = 1
	}
	if a.Audit != nil && len(legal) > 1 {
		a.Audit.observe(legal, probabilities)
	}
	value, choice := a.Random.Float64(), legal[len(legal)-1]
	weights := make([]float64, len(legal))
	for i, action := range legal {
		weights[i] = probabilities[action]
	}
	for _, action := range legal {
		value -= probabilities[action]
		if value < 0 {
			choice = action
			break
		}
	}
	if a.Record && len(legal) > 1 {
		a.Steps = append(a.Steps, neural.PolicyStep{Input: input, Legal: legal, Action: choice, Probabilities: weights, Hidden: hidden})
	}
	if len(legal) > 1 {
		a.Decisions++
	}
	return choice
}
