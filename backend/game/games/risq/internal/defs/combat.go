package defs

import (
	"fmt"
	"github.com/gin-gonic/gin"
)

type AttackType uint8

const (
	AttackType_NONE AttackType = iota
	AttackType_BLUNT
	AttackType_PIERCING
	AttackType_MAGIC
	AttackType_BLUNT_PIERCING
	AttackType_PIERCING_MAGIC
	AttackType_MAGIC_BLUNT
	AttackType_BLUNT_PIERCING_MAGIC
)

// The 9 attack/defense/penetration deltas a tech or bonus config can grant, as one unit so callers
// stop copying all 9 fields by hand at every config-parsing, application, and summation site.
type CombatBonus struct {
	Attack_blunt         int
	Attack_piercing      int
	Attack_magic         int
	Defense_blunt        int
	Defense_piercing     int
	Defense_magic        int
	Penetration_blunt    int
	Penetration_piercing int
	Penetration_magic    int
}

func (b CombatBonus) Add(other CombatBonus) CombatBonus {
	return CombatBonus{
		Attack_blunt:         b.Attack_blunt + other.Attack_blunt,
		Attack_piercing:      b.Attack_piercing + other.Attack_piercing,
		Attack_magic:         b.Attack_magic + other.Attack_magic,
		Defense_blunt:        b.Defense_blunt + other.Defense_blunt,
		Defense_piercing:     b.Defense_piercing + other.Defense_piercing,
		Defense_magic:        b.Defense_magic + other.Defense_magic,
		Penetration_blunt:    b.Penetration_blunt + other.Penetration_blunt,
		Penetration_piercing: b.Penetration_piercing + other.Penetration_piercing,
		Penetration_magic:    b.Penetration_magic + other.Penetration_magic,
	}
}

func (b CombatBonus) ToFrontend() gin.H {
	return gin.H{
		"attack_blunt":         b.Attack_blunt,
		"attack_piercing":      b.Attack_piercing,
		"attack_magic":         b.Attack_magic,
		"defense_blunt":        b.Defense_blunt,
		"defense_piercing":     b.Defense_piercing,
		"defense_magic":        b.Defense_magic,
		"penetration_blunt":    b.Penetration_blunt,
		"penetration_piercing": b.Penetration_piercing,
		"penetration_magic":    b.Penetration_magic,
	}
}

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

type TargetCategory uint8

const (
	TargetCategory_NONE TargetCategory = iota
	TargetCategory_ECONOMIC
	TargetCategory_MILITARY
	TargetCategory_BUILDING
	TargetCategory_END
)

type TargetType struct {
	Orderable_type OrderableType
	Object_type    int // UnitType when orderable_type is OrderableType_UNIT; 0 means "any" within that orderable_type
}

func (tt TargetType) Matches(actual TargetType) bool {
	if tt.Orderable_type != actual.Orderable_type {
		return false
	}
	return tt.Object_type == 0 || tt.Object_type == actual.Object_type
}

func parseTargetType(s string) (TargetType, error) {
	switch s {
	case "building":
		return TargetType{Orderable_type: OrderableType_BUILDING}, nil
	case "unit":
		return TargetType{Orderable_type: OrderableType_UNIT}, nil
	default:
		unit_type, err := parseUnitType(s)
		if err != nil {
			return TargetType{}, fmt.Errorf("unknown target_type %q", s)
		}
		return TargetType{Orderable_type: OrderableType_UNIT, Object_type: int(unit_type)}, nil
	}
}

func targetTypeOf(orderable_type OrderableType, unit_type UnitType) TargetType {
	if orderable_type == OrderableType_UNIT {
		return TargetType{Orderable_type: OrderableType_UNIT, Object_type: int(unit_type)}
	}
	return TargetType{Orderable_type: orderable_type}
}
