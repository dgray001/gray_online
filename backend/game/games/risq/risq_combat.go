package risq

import (
	"math"

	"github.com/dgray001/gray_online/game/game_utils"
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

type AttackedByObjectType uint8

const (
	AttackedByUnit AttackedByObjectType = iota
	AttackedByBuilding
)

type RisqDamageEvent struct {
	tick          uint16
	attacker_id   uint64
	attacker_type AttackedByObjectType
	damage        float64
	damage_type   AttackType
}

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

// Applies a unit's attack against a building, deleting it and logging the raze/loss if it dies
func (r *GameRisq) unitAttackBuilding(attacker *RisqUnit, target *RisqBuilding) {
	was_alive := target.cs.health > 0
	damage := combatDamage(&attacker.cs, &target.cs, attacker.intent.intent_cost)
	target.cs.addHealth(-damage)
	target.attacked_by = append(target.attacked_by, RisqDamageEvent{tick: r.current_tick, attacker_id: attacker.internal_id, attacker_type: AttackedByUnit, damage: damage, damage_type: attacker.cs.attack_type})
	if !was_alive || target.cs.health > 0 {
		return
	}
	space := target.zone.space.coordinate
	zone := target.zone.coordinate
	r.players[attacker.player_id].report.recordCombat(RisqCombatEvent{tick: r.current_tick, kind: CombatEvent_BuildingRazed,
		self_player: attacker.player_id, other_player: target.player_id, target_id: uint64(target.building_id), space: space, zone: zone, damage: damage})
	r.players[target.player_id].report.recordCombat(RisqCombatEvent{tick: r.current_tick, kind: CombatEvent_BuildingLost,
		self_player: target.player_id, other_player: attacker.player_id, target_id: uint64(target.building_id), space: space, zone: zone, damage: damage})
	r.players[attacker.player_id].razes++
	r.players[target.player_id].buildings_lost++
	target.deleted = true
}

func (r *GameRisq) unitAttackUnit(attacker *RisqUnit, target *RisqUnit) {
	was_alive := target.cs.health > 0
	damage := combatDamage(&attacker.cs, &target.cs, attacker.intent.intent_cost)
	target.cs.addHealth(-damage)
	target.attacked_by = append(target.attacked_by, RisqDamageEvent{tick: r.current_tick, attacker_id: attacker.internal_id, attacker_type: AttackedByUnit, damage: damage, damage_type: attacker.cs.attack_type})
	util.DebugLog.Printf("combat tick=%d: unit %d (player %d, stamina %d) hits unit %d (player %d) for %.2f, health now %.2f/%d",
		r.current_tick, attacker.internal_id, attacker.player_id, attacker.intent.intent_cost,
		target.internal_id, target.player_id, damage, target.cs.health, target.cs.max_health)
	if !was_alive || target.cs.health > 0 {
		return
	}
	space := target.zone.space.coordinate
	zone := target.zone.coordinate
	r.players[attacker.player_id].report.recordCombat(RisqCombatEvent{tick: r.current_tick, kind: CombatEvent_UnitKilled,
		self_player: attacker.player_id, other_player: target.player_id, target_id: uint64(target.unit_id), space: space, zone: zone, damage: damage})
	r.players[target.player_id].report.recordCombat(RisqCombatEvent{tick: r.current_tick, kind: CombatEvent_UnitLost,
		self_player: target.player_id, other_player: attacker.player_id, target_id: uint64(target.unit_id), space: space, zone: zone, damage: damage})
	r.players[attacker.player_id].kills++
	r.players[target.player_id].units_lost++
	target.deleted = true
}

// Returns the lowest-internal_id enemy unit in the zone, or nil
func zoneEnemyUnit(zone *RisqZone, player_id int) *RisqUnit {
	var best *RisqUnit
	for _, u := range zone.units {
		if u.deleted || u.player_id == player_id {
			continue
		}
		if best == nil || u.internal_id < best.internal_id {
			best = u
		}
	}
	return best
}

func zoneEnemyBuilding(zone *RisqZone, player_id int) *RisqBuilding {
	if zone.building == nil || zone.building.deleted || zone.building.player_id == player_id {
		return nil
	}
	return zone.building
}

func zoneHasEnemy(zone *RisqZone, player_id int) bool {
	return zoneEnemyUnit(zone, player_id) != nil || zoneEnemyBuilding(zone, player_id) != nil
}

func spaceHasEnemy(space *RisqSpace, player_id int) bool {
	for _, row := range space.zones {
		for _, zone := range row {
			if zoneHasEnemy(zone, player_id) {
				return true
			}
		}
	}
	return false
}

// Returns the zone-hop distance between two zones of the same space via BFS
func zoneDistanceWithinSpace(from *RisqZone, to *RisqZone) int {
	if from == to {
		return 0
	}
	space := from.space
	visited := map[*RisqZone]bool{from: true}
	frontier := []*RisqZone{from}
	for dist := 1; len(frontier) > 0; dist++ {
		next := []*RisqZone{}
		for _, z := range frontier {
			for _, adj := range z.adjacent_zones {
				if adj.space != space || visited[adj] {
					continue
				}
				if adj == to {
					return dist
				}
				visited[adj] = true
				next = append(next, adj)
			}
		}
		frontier = next
	}
	return -1
}

type TargetCategory uint8

const (
	TargetCategory_NONE TargetCategory = iota
	TargetEconomic
	TargetMilitary
	TargetBuilding
)

func targetCategoryOf(u *RisqUnit) TargetCategory {
	if isEconomicUnit(u.unit_id) {
		return TargetEconomic
	}
	return TargetMilitary
}

type categoryBest struct {
	units         map[TargetCategory]*RisqUnit
	unit_dist     map[TargetCategory]int
	building      *RisqBuilding
	building_dist int
}

func newCategoryBest() *categoryBest {
	return &categoryBest{units: make(map[TargetCategory]*RisqUnit), unit_dist: make(map[TargetCategory]int)}
}

func (c *categoryBest) considerUnit(target *RisqUnit, dist int) {
	cat := targetCategoryOf(target)
	if existing, ok := c.units[cat]; !ok || dist < c.unit_dist[cat] || (dist == c.unit_dist[cat] && target.internal_id < existing.internal_id) {
		c.units[cat] = target
		c.unit_dist[cat] = dist
	}
}

func (c *categoryBest) considerBuilding(target *RisqBuilding, dist int) {
	if c.building == nil || dist < c.building_dist || (dist == c.building_dist && target.internal_id < c.building.internal_id) {
		c.building = target
		c.building_dist = dist
	}
}

func (c *categoryBest) pick(priority []TargetCategory) (*RisqUnit, *RisqBuilding) {
	ranked := make(map[TargetCategory]bool, len(priority))
	for _, cat := range priority {
		if cat == TargetCategory_NONE || ranked[cat] {
			continue
		}
		ranked[cat] = true
		if cat == TargetBuilding {
			if c.building != nil {
				return nil, c.building
			}
		} else if target, ok := c.units[cat]; ok {
			return target, nil
		}
	}
	var best_unit *RisqUnit
	best_dist := -1
	for cat, target := range c.units {
		if ranked[cat] {
			continue
		}
		if best_unit == nil || c.unit_dist[cat] < best_dist || (c.unit_dist[cat] == best_dist && target.internal_id < best_unit.internal_id) {
			best_unit, best_dist = target, c.unit_dist[cat]
		}
	}
	if !ranked[TargetBuilding] && c.building != nil && (best_unit == nil || c.building_dist < best_dist) {
		return nil, c.building
	}
	return best_unit, nil
}

func zoneAttackTarget(zone *RisqZone, player_id int, priority []TargetCategory) (*RisqUnit, *RisqBuilding) {
	best := newCategoryBest()
	for _, target := range zone.units {
		if target.deleted || target.player_id == player_id {
			continue
		}
		best.considerUnit(target, 0)
	}
	if target := zoneEnemyBuilding(zone, player_id); target != nil {
		best.considerBuilding(target, 0)
	}
	return best.pick(priority)
}

func spaceAttackTarget(u *RisqUnit, space *RisqSpace) (*RisqUnit, *RisqBuilding) {
	best := newCategoryBest()
	for _, row := range space.zones {
		for _, zone := range row {
			dist := zoneDistanceWithinSpace(u.zone, zone)
			for _, target := range zone.units {
				if target.deleted || target.player_id == u.player_id {
					continue
				}
				best.considerUnit(target, dist)
			}
			if target := zoneEnemyBuilding(zone, u.player_id); target != nil {
				best.considerBuilding(target, dist)
			}
		}
	}
	return best.pick(u.target_priority)
}

func nearbyAttackTarget(u *RisqUnit, risq *GameRisq, space_radius uint) (*RisqUnit, *RisqBuilding) {
	best := newCategoryBest()
	own_space := u.zone.space
	for _, row := range risq.spaces {
		for _, space := range row {
			space_dist := game_utils.AxialDistance(own_space.coordinate, space.coordinate)
			if space_dist > space_radius || space.getVisibility(u.player_id) < VisibilityGood {
				continue
			}
			for _, zone_row := range space.zones {
				for _, zone := range zone_row {
					dist := int(space_dist) * 6
					if space == own_space {
						dist = zoneDistanceWithinSpace(u.zone, zone)
					}
					for _, target := range zone.units {
						if target.deleted || target.player_id == u.player_id {
							continue
						}
						best.considerUnit(target, dist)
					}
					if target := zoneEnemyBuilding(zone, u.player_id); target != nil {
						best.considerBuilding(target, dist)
					}
				}
			}
		}
	}
	return best.pick(u.target_priority)
}
