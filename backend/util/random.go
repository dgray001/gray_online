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
