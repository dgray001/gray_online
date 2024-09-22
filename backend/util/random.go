package util

import "math/rand"

// Returns a boolean indicating whether the random chance occurred
func RandomChance(chance float64) bool {
	return rand.Float64() <= chance
}

// Returns an int in the range [min, max], or [max, min] if min > max
func RandomInt(min int, max int) int {
	if min > max {
		return RandomInt(max, min)
	}
	return min + int(rand.Int63n(1+int64(max-min)))
}

// Returns an float in the range [min, max], or [max, min] if min > max
func RandomFloat(min float64, max float64) float64 {
	if min > max {
		return RandomFloat(max, min)
	}
	return min + (max-min)*rand.Float64()
}
