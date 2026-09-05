package risq

import (
	"math"

	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

const combatRateStaminaBase = 10.0
const combatFloorPerStamina = 0.1
const combatKneeRatio = 0.5

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

type RisqCombatStats struct {
	health               float64
	max_health           int
	attack_type          AttackType
	attack_blunt         int
	attack_piercing      int
	attack_magic         int
	defense_blunt        int
	defense_piercing     int
	defense_magic        int
	penetration_blunt    int
	penetration_piercing int
	penetration_magic    int
}

func createRisqCombatStats() RisqCombatStats {
	return RisqCombatStats{
		health:               1,
		max_health:           1,
		attack_type:          AttackType_NONE,
		attack_blunt:         0,
		attack_piercing:      0,
		attack_magic:         0,
		defense_blunt:        0,
		defense_piercing:     0,
		defense_magic:        0,
		penetration_blunt:    0,
		penetration_piercing: 0,
		penetration_magic:    0,
	}
}

func (c *RisqCombatStats) setHealthRatio(ratio float64) {
	c.health = util.Clamp(ratio, 0.0, 1.0) * float64(c.max_health)
}

func (c *RisqCombatStats) addHealth(amount float64) {
	c.health = util.Clamp(c.health+amount, 0, float64(c.max_health))
}

func (c *RisqCombatStats) setMaxHealth(max_health int) {
	if max_health < 1 {
		return
	}
	ratio := c.health / float64(c.max_health)
	c.max_health = max_health
	c.health = ratio * float64(max_health)
}

func (cs *RisqCombatStats) toFrontend() gin.H {
	return gin.H{
		"health":               cs.health,
		"max_health":           cs.max_health,
		"attack_type":          cs.attack_type,
		"attack_blunt":         cs.attack_blunt,
		"attack_piercing":      cs.attack_piercing,
		"attack_magic":         cs.attack_magic,
		"defense_blunt":        cs.defense_blunt,
		"defense_piercing":     cs.defense_piercing,
		"defense_magic":        cs.defense_magic,
		"penetration_blunt":    cs.penetration_blunt,
		"penetration_piercing": cs.penetration_piercing,
		"penetration_magic":    cs.penetration_magic,
	}
}

func effectiveDefense(defense int, penetration int) float64 {
	return float64(defense) * (1 - float64(penetration)/100)
}

// Returns the effective attack and defense of the attacker and defender
func combatTotals(attacker *RisqCombatStats, defender *RisqCombatStats) (float64, float64) {
	attack := 0.0
	defense := 0.0
	switch attacker.attack_type {
	case AttackType_BLUNT, AttackType_BLUNT_PIERCING, AttackType_MAGIC_BLUNT, AttackType_BLUNT_PIERCING_MAGIC:
		attack += float64(attacker.attack_blunt)
		defense += effectiveDefense(defender.defense_blunt, attacker.penetration_blunt)
	}
	switch attacker.attack_type {
	case AttackType_PIERCING, AttackType_BLUNT_PIERCING, AttackType_PIERCING_MAGIC, AttackType_BLUNT_PIERCING_MAGIC:
		attack += float64(attacker.attack_piercing)
		defense += effectiveDefense(defender.defense_piercing, attacker.penetration_piercing)
	}
	switch attacker.attack_type {
	case AttackType_MAGIC, AttackType_PIERCING_MAGIC, AttackType_MAGIC_BLUNT, AttackType_BLUNT_PIERCING_MAGIC:
		attack += float64(attacker.attack_magic)
		defense += effectiveDefense(defender.defense_magic, attacker.penetration_magic)
	}
	return attack, defense
}

// Calculate damage for one tick based on input stamina
func combatDamage(attacker *RisqCombatStats, defender *RisqCombatStats, stamina_spent int) float64 {
	attack, defense := combatTotals(attacker, defender)
	if attack <= 0 {
		return 0
	}
	ratio := defense / attack
	var reference float64
	if ratio <= combatKneeRatio {
		reference = attack - defense
	} else {
		reference = attack * math.Pow(4, -ratio)
	}
	damage := reference * float64(stamina_spent) / combatRateStaminaBase
	floor := combatFloorPerStamina * float64(stamina_spent)
	if damage < floor {
		damage = floor
	}
	return damage
}
