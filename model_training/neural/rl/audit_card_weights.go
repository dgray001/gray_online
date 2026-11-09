package rl

import "fiddlesticks.live/neural"

func auditCardWeights(audit *PolicyAudit, weights []float64) {
	policy := samplerProbabilities(weights)
	legal, probabilities := make([]int, len(weights)), make([]float64, neural.Outputs)
	for i, probability := range policy {
		legal[i] = 27 + i
		probabilities[27+i] = probability
	}
	audit.observe(legal, probabilities)
}
