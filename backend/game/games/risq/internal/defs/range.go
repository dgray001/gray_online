package defs

import "fmt"

type RisqRange uint8

const (
	RisqRange_NONE RisqRange = iota
	RisqRange_ZONE
	RisqRange_SPACE
	RisqRange_ADJACENT
	RisqRange_SECONDARY
	RisqRange_END
)

func (r RisqRange) SpaceRadius() (uint, bool) {
	switch r {
	case RisqRange_SPACE:
		return 0, true
	case RisqRange_ADJACENT:
		return 1, true
	case RisqRange_SECONDARY:
		return 2, true
	default:
		return 0, false
	}
}

func parseRange(s string) (RisqRange, error) {
	switch s {
	case "", "zone":
		return RisqRange_ZONE, nil
	case "space":
		return RisqRange_SPACE, nil
	case "adjacent":
		return RisqRange_ADJACENT, nil
	case "secondary":
		return RisqRange_SECONDARY, nil
	default:
		return RisqRange_NONE, fmt.Errorf("unknown range %q", s)
	}
}
