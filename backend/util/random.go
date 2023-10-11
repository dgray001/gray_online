package util

import "math/rand"

func RandomChance(chance float64) bool {
	return rand.Float64() <= chance
}
