package risq

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

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

type RisqUnit struct {
	deleted              bool
	internal_id          uint64
	player_id            int
	unit_id              uint32
	zone                 *RisqZone
	health               int
	max_health           int
	speed                int
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

func createRisqUnit(internal_id uint64, unit_id uint32, player_id int) *RisqUnit {
	unit := RisqUnit{
		deleted:              false,
		internal_id:          internal_id,
		player_id:            player_id,
		unit_id:              unit_id,
		max_health:           1,
		speed:                1,
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
	switch unit_id {
	case 1: // villager
		unit.max_health = 25
		unit.attack_type = AttackType_BLUNT
		unit.attack_blunt = 3
	default:
		fmt.Fprintln(os.Stderr, "Creating unknown unit id: ", unit_id)
	}
	unit.health = unit.max_health
	return &unit
}

func (u *RisqUnit) toFrontend() gin.H {
	unit := gin.H{
		"internal_id":          u.internal_id,
		"player_id":            u.player_id,
		"unit_id":              u.unit_id,
		"max_health":           u.max_health,
		"speed":                u.speed,
		"attack_type":          u.attack_type,
		"attack_blunt":         u.attack_blunt,
		"attack_piercing":      u.attack_piercing,
		"attack_magic":         u.attack_magic,
		"defense_blunt":        u.defense_blunt,
		"defense_piercing":     u.defense_piercing,
		"defense_magic":        u.defense_magic,
		"penetration_blunt":    u.penetration_blunt,
		"penetration_piercing": u.penetration_piercing,
		"penetration_magic":    u.penetration_magic,
	}
	if u.zone != nil {
		unit["zone_coordinate"] = u.zone.coordinate.ToFrontend()
		if u.zone.space != nil {
			unit["space_coordinate"] = u.zone.space.coordinate.ToFrontend()
		}
	}
	return unit
}
