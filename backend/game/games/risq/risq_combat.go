package risq

import "github.com/gin-gonic/gin"

type AttackType int8

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
	health               int
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

func (c *RisqCombatStats) setMaxHealth(max_health int) {
	if max_health < 1 {
		return
	}
	ratio := float64(c.health) / float64(c.max_health)
	c.max_health = max_health
	c.health = int(ratio*float64(max_health) + 0.5)
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
