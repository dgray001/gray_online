package defs

import "math"

func ResolveSetupSetting[T ~uint8](raw any, fallback T, maximum T) T {
	value, ok := raw.(float64)
	if !ok || math.IsNaN(value) || value < 1 || value > float64(maximum) || value != math.Trunc(value) {
		return fallback
	}
	return T(value)
}
