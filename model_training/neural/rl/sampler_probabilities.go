package rl

import "math"

func samplerProbabilities(weights []float64) []float64 {
	result, total := make([]float64, len(weights)), 0.0
	for _, weight := range weights {
		total += weight
	}
	if total == 0 {
		value := 0.0
		for i, weight := range weights {
			if value < weight {
				result[i] = 1
				return result
			}
			value -= weight
		}
		result[0] = 1
		return result
	}
	lower, upper := math.Min(0, total), math.Max(0, total)
	covered, prefix, mass := lower, 0.0, 0.0
	for i, weight := range weights {
		prefix += weight
		threshold := math.Max(lower, math.Min(upper, prefix))
		if threshold > covered {
			result[i] = (threshold - covered) / (upper - lower)
			mass += result[i]
			covered = threshold
		}
	}
	result[0] += math.Max(0, 1-mass)
	return result
}
