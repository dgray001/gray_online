package defs

import "math"

func ResolveSetupSetting[T ~uint8](raw any, fallback T, maximum T) T {
	value, ok := raw.(float64)
	if !ok || math.IsNaN(value) || value < 1 || value > float64(maximum) || value != math.Trunc(value) {
		return fallback
	}
	return T(value)
}

type StartBalance uint8

const (
	StartBalance_NONE StartBalance = iota
	StartBalance_RANDOM
	StartBalance_MIRRORED
)

type MapResourceLevel uint8

const (
	MapResourceLevel_NONE MapResourceLevel = iota
	MapResourceLevel_SPARSE
	MapResourceLevel_STANDARD
	MapResourceLevel_RICH
	MapResourceLevel_ABUNDANT
)

type StartingDevelopment uint8

const (
	StartingDevelopment_NONE StartingDevelopment = iota
	StartingDevelopment_NOMAD
	StartingDevelopment_STANDARD
	StartingDevelopment_ESTABLISHED
	StartingDevelopment_ADVANCED
)

type StartingStockpile uint8

const (
	StartingStockpile_NONE StartingStockpile = iota
	StartingStockpile_LOW
	StartingStockpile_STANDARD
	StartingStockpile_HIGH
	StartingStockpile_DEATHMATCH
)

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
