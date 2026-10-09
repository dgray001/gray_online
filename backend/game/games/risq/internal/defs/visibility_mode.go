package defs

import "math"

type VisibilityMode uint8

const (
	VisibilityMode_NONE VisibilityMode = iota
	VisibilityMode_DEFAULT
	VisibilityMode_EXPLORED
	VisibilityMode_ALL_VISIBLE
)

func ResolveVisibilityMode(raw any) VisibilityMode {
	value, ok := raw.(float64)
	if !ok || value < 1 || value > float64(VisibilityMode_ALL_VISIBLE) || value != math.Trunc(value) {
		return VisibilityMode_DEFAULT
	}
	return VisibilityMode(value)
}

func (v VisibilityMode) MinimumVision() VisibilityLevel {
	switch v {
	case VisibilityMode_EXPLORED:
		return VisibilityFog
	case VisibilityMode_ALL_VISIBLE:
		return VisibilityGood
	default:
		return VisibilityUnexplored
	}
}
