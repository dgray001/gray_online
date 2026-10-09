package defs

import "math"

type MapSize uint8

const (
	MapSize_NONE MapSize = iota
	MapSize_MINUSCULE
	MapSize_TINY
	MapSize_SMALLER
	MapSize_SMALL
	MapSize_MEDIUM
	MapSize_LARGE
	MapSize_LARGER
	MapSize_HUGE
	MapSize_GIGANTIC
)

func DefaultMapSize(num_players int) MapSize {
	switch {
	case num_players <= 2:
		return MapSize_MINUSCULE
	case num_players <= 8:
		return MapSize(num_players - 1)
	case num_players <= 10:
		return MapSize_HUGE
	default:
		return MapSize_GIGANTIC
	}
}

func ResolveMapSize(raw any, num_players int) MapSize {
	value, ok := raw.(float64)
	if !ok || value < 1 || value > float64(MapSize_GIGANTIC) || value != math.Trunc(value) {
		return DefaultMapSize(num_players)
	}
	return MapSize(value)
}
