package defs

import "github.com/gin-gonic/gin"

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
