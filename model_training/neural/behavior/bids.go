package behavior

import "math"

func bidPolicy(mean float64, count int) []float64 {
	result := make([]float64, count)
	lower := int(math.Floor(mean))
	result[lower] = 1 - (mean - float64(lower))
	if lower+1 < count {
		result[lower+1] = mean - float64(lower)
	}
	return result
}
