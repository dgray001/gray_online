package risq

import (
	"fmt"
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

func rangeCovers(from *RisqZone, target *RisqZone, attack_range RisqRange) bool {
	if from == nil || target == nil {
		return false
	}
	space_range, ranged := attack_range.spaceRadius()
	if !ranged {
		return from == target
	}
	return game_utils.AxialDistance(from.space.coordinate, target.space.coordinate) <= space_range
}

func (u *RisqUnit) inAttackRange(target *RisqZone) bool {
	return rangeCovers(u.zone, target, u.attack_range)
}

func (b *RisqBuilding) inAttackRange(target *RisqZone) bool {
	return rangeCovers(b.zone, target, b.attack_range)
}

type RisqDamageEvent struct {
	tick          uint16
	attacker_id   uint64
	attacker_type OrderableType
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

func attackTypeHasBlunt(t AttackType) bool {
	switch t {
	case AttackType_BLUNT, AttackType_BLUNT_PIERCING, AttackType_MAGIC_BLUNT, AttackType_BLUNT_PIERCING_MAGIC:
		return true
	default:
		return false
	}
}

func attackTypeHasPiercing(t AttackType) bool {
	switch t {
	case AttackType_PIERCING, AttackType_BLUNT_PIERCING, AttackType_PIERCING_MAGIC, AttackType_BLUNT_PIERCING_MAGIC:
		return true
	default:
		return false
	}
}

func attackTypeHasMagic(t AttackType) bool {
	switch t {
	case AttackType_MAGIC, AttackType_PIERCING_MAGIC, AttackType_MAGIC_BLUNT, AttackType_BLUNT_PIERCING_MAGIC:
		return true
	default:
		return false
	}
}

func (cs *RisqCombatStats) totalAttack() int {
	total := 0
	if attackTypeHasBlunt(cs.attack_type) {
		total += cs.attack_blunt
	}
	if attackTypeHasPiercing(cs.attack_type) {
		total += cs.attack_piercing
	}
	if attackTypeHasMagic(cs.attack_type) {
		total += cs.attack_magic
	}
	return total
}

// Returns the effective attack and defense of the attacker and defender
func combatTotals(attacker *RisqCombatStats, defender *RisqCombatStats) (float64, float64) {
	defense := 0.0
	if attackTypeHasBlunt(attacker.attack_type) {
		defense += effectiveDefense(defender.defense_blunt, attacker.penetration_blunt)
	}
	if attackTypeHasPiercing(attacker.attack_type) {
		defense += effectiveDefense(defender.defense_piercing, attacker.penetration_piercing)
	}
	if attackTypeHasMagic(attacker.attack_type) {
		defense += effectiveDefense(defender.defense_magic, attacker.penetration_magic)
	}
	return float64(attacker.totalAttack()), defense
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

type Attackable interface {
	Orderable
	playerId() int
	combatStats(r *GameRisq, other Orderable, attacking bool) RisqCombatStats
	isAlive() bool
	applyDamage(event RisqDamageEvent)
	recordDeath(r *GameRisq, attacker Attackable, damage float64)
}

// Shared attack resolution for any unit/building attacker-target pair
func (r *GameRisq) resolveAttack(attacker Attackable, target Attackable, stamina_cost int) {
	was_alive := target.isAlive()
	attacker_cs := attacker.combatStats(r, target, true)
	target_cs := target.combatStats(r, attacker, false)
	damage := combatDamage(&attacker_cs, &target_cs, stamina_cost)
	target.applyDamage(RisqDamageEvent{tick: r.current_tick, attacker_id: attacker.internalId(), attacker_type: attacker.OrderableType(), damage: damage, damage_type: attacker_cs.attack_type})
	util.DebugLog.Printf("combat tick=%d: %d (player %d, stamina %d) hits %d (player %d) for %.2f",
		r.current_tick, attacker.internalId(), attacker.playerId(), stamina_cost, target.internalId(), target.playerId(), damage)
	if !was_alive || target.isAlive() {
		return
	}
	target.recordDeath(r, attacker, damage)
}

func (r *GameRisq) unitAttack(attacker *RisqUnit, target Attackable) {
	r.resolveAttack(attacker, target, attacker.intent.intent_cost)
}

func (r *GameRisq) buildingAttack(attacker *RisqBuilding, target Attackable) {
	r.resolveAttack(attacker, target, attacker.intent.intent_cost)
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
	TargetCategory_ECONOMIC
	TargetCategory_MILITARY
	TargetCategory_BUILDING
	TargetCategory_END
)

func targetCategoryOf(u *RisqUnit) TargetCategory {
	if u.unitType() == UnitType_ECONOMIC {
		return TargetCategory_ECONOMIC
	}
	return TargetCategory_MILITARY
}

type TargetType struct {
	orderable_type OrderableType
	object_type    int // UnitType when orderable_type is OrderableType_UNIT; 0 means "any" within that orderable_type
}

func targetTypeOf(orderable_type OrderableType, unit_type UnitType) TargetType {
	if orderable_type == OrderableType_UNIT {
		return TargetType{orderable_type: OrderableType_UNIT, object_type: int(unit_type)}
	}
	return TargetType{orderable_type: orderable_type}
}

func (tt TargetType) matches(actual TargetType) bool {
	if tt.orderable_type != actual.orderable_type {
		return false
	}
	return tt.object_type == 0 || tt.object_type == actual.object_type
}

func parseTargetType(s string) (TargetType, error) {
	switch s {
	case "building":
		return TargetType{orderable_type: OrderableType_BUILDING}, nil
	case "unit":
		return TargetType{orderable_type: OrderableType_UNIT}, nil
	default:
		unit_type, err := parseUnitType(s)
		if err != nil {
			return TargetType{}, fmt.Errorf("unknown target_type %q", s)
		}
		return TargetType{orderable_type: OrderableType_UNIT, object_type: int(unit_type)}, nil
	}
}

func otherUnitIdentity(other Orderable) (uint32, UnitType) {
	if unit, ok := other.(*RisqUnit); ok {
		return unit.unit_id, unit.unitType()
	}
	return 0, UnitType_NONE
}

func (r *GameRisq) effectiveCombatStats(u *RisqUnit, other Orderable, attacking bool) RisqCombatStats {
	cs := u.cs
	other_unit_id, other_unit_type := otherUnitIdentity(other)
	bonus := sumTargetedBonus(u.unit_id, u.unitType(), r.players[u.player_id].researchedTechIds(), other_unit_id, other_unit_type, other.OrderableType())
	if attacking {
		cs.attack_blunt += bonus.bonus_attack_blunt
		cs.attack_piercing += bonus.bonus_attack_piercing
		cs.attack_magic += bonus.bonus_attack_magic
		cs.penetration_blunt += bonus.bonus_penetration_blunt
		cs.penetration_piercing += bonus.bonus_penetration_piercing
		cs.penetration_magic += bonus.bonus_penetration_magic
	} else {
		cs.defense_blunt += bonus.bonus_defense_blunt
		cs.defense_piercing += bonus.bonus_defense_piercing
		cs.defense_magic += bonus.bonus_defense_magic
	}
	return cs
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
		if cat == TargetCategory_BUILDING {
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
	if !ranked[TargetCategory_BUILDING] && c.building != nil && (best_unit == nil || c.building_dist < best_dist) {
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
	near, far := newCategoryBest(), newCategoryBest()
	for _, row := range space.zones {
		for _, zone := range row {
			dist := zoneDistanceWithinSpace(u.zone, zone)
			bucket := far
			if u.inAttackRange(zone) {
				bucket = near
			}
			for _, target := range zone.units {
				if target.deleted || target.player_id == u.player_id {
					continue
				}
				bucket.considerUnit(target, dist)
			}
			if target := zoneEnemyBuilding(zone, u.player_id); target != nil {
				bucket.considerBuilding(target, dist)
			}
		}
	}
	if target, building := near.pick(u.target_priority); target != nil || building != nil {
		return target, building
	}
	return far.pick(u.target_priority)
}

func nearbyAttackTarget(own_zone *RisqZone, player_id int, target_priority []TargetCategory, in_range func(*RisqZone) bool, risq *GameRisq, space_radius uint) (*RisqUnit, *RisqBuilding) {
	near, far := newCategoryBest(), newCategoryBest()
	own_space := own_zone.space
	for _, row := range risq.spaces {
		for _, space := range row {
			space_dist := game_utils.AxialDistance(own_space.coordinate, space.coordinate)
			if space_dist > space_radius || space.getVisibility(player_id) < VisibilityGood {
				continue
			}
			for _, zone_row := range space.zones {
				for _, zone := range zone_row {
					dist := int(space_dist) * 6
					if space == own_space {
						dist = zoneDistanceWithinSpace(own_zone, zone)
					}
					bucket := far
					if in_range(zone) {
						bucket = near
					}
					for _, target := range zone.units {
						if target.deleted || target.player_id == player_id {
							continue
						}
						bucket.considerUnit(target, dist)
					}
					if target := zoneEnemyBuilding(zone, player_id); target != nil {
						bucket.considerBuilding(target, dist)
					}
				}
			}
		}
	}
	if target, building := near.pick(target_priority); target != nil || building != nil {
		return target, building
	}
	return far.pick(target_priority)
}
