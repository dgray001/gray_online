package neural

import "math"

func bidDistribution(bid float64, count int) []float64 {
	distribution := make([]float64, count)
	lower := int(math.Floor(bid))
	distribution[lower] = 1 - (bid - float64(lower))
	if lower+1 < count {
		distribution[lower+1] = bid - float64(lower)
	}
	return distribution
}

func normalized(weights []float64) []float64 {
	total := 0.0
	for _, weight := range weights {
		total += weight
	}
	result := make([]float64, len(weights))
	for i, weight := range weights {
		result[i] = weight / total
	}
	return result
}
