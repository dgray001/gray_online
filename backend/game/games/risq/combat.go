package risq

import (
	"math"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

const combatRateStaminaBase = 10.0
const combatFloorPerStamina = 0.1
const combatKneeRatio = 0.5

func rangeCovers(from *RisqZone, target *RisqZone, attack_range defs.RisqRange) bool {
	if from == nil || target == nil {
		return false
	}
	space_range, ranged := attack_range.SpaceRadius()
	if !ranged {
		return from == target
	}
	return game_utils.AxialDistance(from.space.coordinate, target.space.coordinate) <= space_range
}

type RisqDamageEvent struct {
	tick          uint16
	attacker_id   uint64
	attacker_type defs.OrderableType
	damage        float64
	damage_type   defs.AttackType
}

type RisqCombatStats struct {
	health               float64
	pending_health_delta float64
	max_health           int
	attack_type          defs.AttackType
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

func (cs *RisqCombatStats) applyAttackBonus(b defs.CombatBonus) {
	cs.attack_blunt += b.Attack_blunt
	cs.attack_piercing += b.Attack_piercing
	cs.attack_magic += b.Attack_magic
	cs.penetration_blunt += b.Penetration_blunt
	cs.penetration_piercing += b.Penetration_piercing
	cs.penetration_magic += b.Penetration_magic
}

func (cs *RisqCombatStats) applyDefenseBonus(b defs.CombatBonus) {
	cs.defense_blunt += b.Defense_blunt
	cs.defense_piercing += b.Defense_piercing
	cs.defense_magic += b.Defense_magic
}

func (cs *RisqCombatStats) applyBonus(b defs.CombatBonus) {
	cs.applyAttackBonus(b)
	cs.applyDefenseBonus(b)
}

func createRisqCombatStats() RisqCombatStats {
	return RisqCombatStats{
		health:               1,
		max_health:           1,
		attack_type:          defs.AttackType_NONE,
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
	c.health = util.Clamp(util.RoundTo(c.health+amount, gatherRoundingPlaces), 0, float64(c.max_health))
}

// Accumulates a health change to be applied once at end-of-tick, so damage and
// healing landing in the same tick net out instead of racing on execution order.
func (c *RisqCombatStats) queueHealth(amount float64) {
	c.pending_health_delta += util.RoundTo(amount, gatherRoundingPlaces)
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

func attackTypeHasBlunt(t defs.AttackType) bool {
	switch t {
	case defs.AttackType_BLUNT, defs.AttackType_BLUNT_PIERCING, defs.AttackType_MAGIC_BLUNT, defs.AttackType_BLUNT_PIERCING_MAGIC:
		return true
	default:
		return false
	}
}

func attackTypeHasPiercing(t defs.AttackType) bool {
	switch t {
	case defs.AttackType_PIERCING, defs.AttackType_BLUNT_PIERCING, defs.AttackType_PIERCING_MAGIC, defs.AttackType_BLUNT_PIERCING_MAGIC:
		return true
	default:
		return false
	}
}

func attackTypeHasMagic(t defs.AttackType) bool {
	switch t {
	case defs.AttackType_MAGIC, defs.AttackType_PIERCING_MAGIC, defs.AttackType_MAGIC_BLUNT, defs.AttackType_BLUNT_PIERCING_MAGIC:
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

func combatTotals(attacker *RisqCombatStats, defender *RisqCombatStats) (attack float64, defense float64) {
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
	// Applies this tick's queued health changes (damage and healing) as one net
	// amount, then records death if that net change was the cause.
	resolveHealthDelta(r *GameRisq)
	currentZone() *RisqZone
}

// Shared attack resolution for any unit/building attacker-target pair. Damage is
// only queued here; resolveHealthDelta applies it (and any healing queued the same
// tick) once resolution's execute phase finishes, so order within the tick can't matter.
func (r *GameRisq) resolveAttack(attacker Attackable, target Attackable, stamina_cost int) {
	attacker_cs := attacker.combatStats(r, target, true)
	target_cs := target.combatStats(r, attacker, false)
	damage := combatDamage(&attacker_cs, &target_cs, stamina_cost)
	target.applyDamage(RisqDamageEvent{tick: r.current_tick, attacker_id: attacker.internalId(), attacker_type: attacker.OrderableType(), damage: damage, damage_type: attacker_cs.attack_type})
	util.DebugLog.Printf("combat tick=%d: %d (player %d, stamina %d) hits %d (player %d) for %.2f",
		r.current_tick, attacker.internalId(), attacker.playerId(), stamina_cost, target.internalId(), target.playerId(), damage)
}

// Deterministically settles same-tick kill credit when several attackers hit one target: lowest internal_id wins.
func lowestIdAttackerEvent(events []RisqDamageEvent, tick uint16) (RisqDamageEvent, bool) {
	var best RisqDamageEvent
	found := false
	for _, event := range events {
		if event.tick != tick {
			continue
		}
		if !found || event.attacker_id < best.attacker_id {
			best = event
			found = true
		}
	}
	return best, found
}

// Resolves a damage event's attacker back to the live actor that dealt it, for death credit.
func (r *GameRisq) resolveAttacker(event RisqDamageEvent) Attackable {
	switch event.attacker_type {
	case defs.OrderableType_UNIT:
		if attacker, ok := r.units[event.attacker_id]; ok {
			return attacker
		}
	case defs.OrderableType_BUILDING:
		if attacker, ok := r.buildings[event.attacker_id]; ok {
			return attacker
		}
	}
	return nil
}

func (r *GameRisq) unitAttack(attacker *RisqUnit, target Attackable) {
	r.resolveAttack(attacker, target, attacker.intent.intent_cost)
}

func (r *GameRisq) buildingAttack(attacker *RisqBuilding, target Attackable) {
	r.resolveAttack(attacker, target, attacker.intent.intent_cost)
}

func otherUnitIdentity(other Orderable) (uint32, defs.UnitType) {
	if unit, ok := other.(*RisqUnit); ok {
		return unit.unit_id, unit.unitType()
	}
	return 0, defs.UnitType_NONE
}

func (r *GameRisq) effectiveCombatStats(u *RisqUnit, other Orderable, attacking bool) RisqCombatStats {
	cs := u.cs
	other_unit_id, other_unit_type := otherUnitIdentity(other)
	bonus := defs.SumTargetedBonus(u.unit_id, u.unitType(), r.players[u.player_id].researchedTechIds(), other_unit_id, other_unit_type, other.OrderableType())
	if attacking {
		cs.applyAttackBonus(bonus)
	} else {
		cs.applyDefenseBonus(bonus)
	}
	return cs
}
