package risq

import (
	"strings"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
)

const aggressivePursuitRadius = 1

func (u *RisqUnit) stanceReachable(target_zone *RisqZone) bool {
	if target_zone == nil || u.zone == nil {
		return false
	}
	if u.inAttackRange(target_zone) {
		return true
	}
	switch u.stance {
	case defs.UnitStance_AGGRESSIVE:
		return u.zone.space.distanceTo(target_zone.space) <= aggressivePursuitRadius
	case defs.UnitStance_DEFENSIVE:
		if u.wallAcrossBorder(target_zone, false) {
			return true
		}
		for _, row := range u.zone.space.zones {
			for _, zone := range row {
				if rangeCovers(zone, target_zone, u.attack_range) {
					return true
				}
			}
		}
		return false
	case defs.UnitStance_STAND_GROUND:
		return u.wallAcrossBorder(target_zone, true)
	default:
		return false
	}
}

func (u *RisqUnit) wallAcrossBorder(target_zone *RisqZone, own_zone_only bool) bool {
	neighbor := target_zone.borderNeighbor()
	if neighbor == nil || !target_zone.knownWallBlocksEntry(neighbor, u.player_id) {
		return false
	}
	if own_zone_only {
		return neighbor == u.zone
	}
	return neighbor.space == u.zone.space
}

func (u *RisqUnit) isAttacking() bool {
	if len(u.order_queue.active_orders) == 0 {
		return false
	}
	switch u.order_queue.active_orders[0].order_type {
	case defs.OrderType_UnitAttackUnit, defs.OrderType_UnitAutoAttackUnit, defs.OrderType_UnitAttackBuilding, defs.OrderType_UnitAutoAttackBuilding, defs.OrderType_UnitAttackZone, defs.OrderType_UnitAttackSpace:
		return true
	default:
		return false
	}
}

// An explicit attack-unit or attack-building order; unlike the stance's own auto, space and zone attacks, the stance never replaces it
func (u *RisqUnit) holdsAssignedAttack() bool {
	if len(u.order_queue.active_orders) == 0 {
		return false
	}
	order_type := u.order_queue.active_orders[0].order_type
	return order_type == defs.OrderType_UnitAttackUnit || order_type == defs.OrderType_UnitAttackBuilding
}

func attackBackDistance(u *RisqUnit, attacker_zone *RisqZone) int {
	return zoneApproachDistance(u.zone, attacker_zone)
}

// Among attackers that hit this tick, applies target_priority then nearest/lowest-id as a tiebreak.
func (u *RisqUnit) reactiveAttackBackTarget(risq *GameRisq) Attackable {
	if !u.attack_back || len(u.attacked_by) == 0 {
		return nil
	}
	latest_tick := u.attacked_by[len(u.attacked_by)-1].tick
	best := newCategoryBest()
	for _, event := range u.attacked_by {
		if event.tick != latest_tick {
			continue
		}
		switch event.attacker_type {
		case defs.OrderableType_UNIT:
			attacker := risq.units[event.attacker_id]
			if attacker == nil || attacker.deleted {
				continue
			}
			if attacker.garrisoned_in != nil {
				building := attacker.garrisoned_in
				if building.deleted || building.zone == nil || !u.stanceReachable(building.zone) {
					continue
				}
				best.considerBuilding(building, attackBackDistance(u, building.zone))
				continue
			}
			if attacker.zone == nil || !u.stanceReachable(attacker.zone) {
				continue
			}
			best.considerUnit(attacker, attackBackDistance(u, attacker.zone))
		case defs.OrderableType_BUILDING:
			attacker := risq.buildings[event.attacker_id]
			if attacker == nil || attacker.deleted || attacker.zone == nil || !u.stanceReachable(attacker.zone) {
				continue
			}
			best.considerBuilding(attacker, attackBackDistance(u, attacker.zone))
		}
	}
	return best.pick(u.target_priority)
}

func (u *RisqUnit) replaceOrder(risq *GameRisq, order_type defs.OrderType, target_id int64, tick_source string) {
	if len(u.order_queue.active_orders) > 0 {
		current := u.order_queue.active_orders[0]
		if current.order_type == order_type && current.target_id == target_id {
			return
		}
	}
	order := createRisqOrder(risq.nextOrderInternalId(), order_type, u.player_id, map[uint64]Orderable{u.internal_id: u}, target_id, true)
	order.tick_source = tick_source
	if !u.orderReceivable(order, risq) {
		return
	}
	risq.addSyntheticOrder(order, risq.players[u.player_id], false)
}

func (r *GameRisq) attackOrderTarget(order *RisqOrder) Attackable {
	if order == nil {
		return nil
	}
	if order.order_type == defs.OrderType_UnitAutoAttackBuilding || order.order_type == defs.OrderType_UnitAttackBuilding || order.order_type == defs.OrderType_BuildingAutoAttackBuilding {
		if target := r.buildings[uint64(order.target_id)]; target != nil {
			return target
		}
		return nil
	}
	if target := r.units[uint64(order.target_id)]; target != nil {
		return target
	}
	return nil
}

func (u *RisqUnit) shouldOverrideAttackBack(old_target Attackable, new_target Attackable) bool {
	if old_target == nil || !old_target.isAlive() || old_target.isDeleted() || old_target.currentZone() == nil {
		return true
	}
	old_zone, new_zone := old_target.currentZone(), new_target.currentZone()
	old_in_range, new_in_range := u.inAttackRange(old_zone), u.inAttackRange(new_zone)
	if new_in_range != old_in_range {
		return new_in_range
	}
	if new_in_range && old_in_range {
		return targetPriorityRank(new_target, u.target_priority) < targetPriorityRank(old_target, u.target_priority)
	}
	old_dist, new_dist := attackBackDistance(u, old_zone), attackBackDistance(u, new_zone)
	if new_dist != old_dist {
		return new_dist < old_dist
	}
	return targetPriorityRank(new_target, u.target_priority) < targetPriorityRank(old_target, u.target_priority)
}

func (u *RisqUnit) resolveStance(risq *GameRisq) {
	if u.zone == nil || u.cs.attack_type == defs.AttackType_NONE {
		return
	}
	if active := u.order_queue.active_orders; len(active) > 0 && active[0].tick_source == "stance" && !targetInPriority(risq.attackOrderTarget(active[0]), u.target_priority) {
		u.cancelOrder(active[0], risq)
	}
	if target := u.reactiveAttackBackTarget(risq); target != nil {
		idle_enough := u.interrupt_current
		if !idle_enough {
			if u.stance == defs.UnitStance_PASSIVE {
				idle_enough = len(u.order_queue.active_orders) == 0
			} else {
				idle_enough = !u.isAttacking()
			}
		}
		if len(u.order_queue.active_orders) > 0 {
			current := u.order_queue.active_orders[0]
			if !targetInPriority(target, u.target_priority) || (current.tick_source == "attack_back" && !u.shouldOverrideAttackBack(risq.attackOrderTarget(current), target)) {
				idle_enough = false
			}
		}
		if idle_enough {
			order_type := attackOrderType(target, defs.OrderType_UnitAutoAttackUnit, defs.OrderType_UnitAutoAttackBuilding)
			u.replaceOrder(risq, order_type, int64(target.internalId()), "attack_back")
		}
		return
	}
	if u.stance == defs.UnitStance_PASSIVE {
		return
	}
	if (!u.interrupt_current && len(u.order_queue.active_orders) > 0) || u.holdsAssignedAttack() {
		return
	}
	space_range, ranged := u.attack_range.SpaceRadius()
	if ranged {
		space_range = max(space_range, u.attack_range.ZoneRadius())
	}
	if u.stance == defs.UnitStance_AGGRESSIVE {
		space_range = max(space_range, aggressivePursuitRadius)
	}
	if u.stance == defs.UnitStance_DEFENSIVE || u.stance == defs.UnitStance_STAND_GROUND {
		space_range = max(space_range, 1)
	}
	target := nearbyTarget(u.zone, u.player_id, u.target_priority, u.stanceReachable, u.inAttackRange, u.canReachNearby, risq, space_range)
	if targetInPriority(target, u.target_priority) {
		order_type := attackOrderType(target, defs.OrderType_UnitAutoAttackUnit, defs.OrderType_UnitAutoAttackBuilding)
		u.replaceOrder(risq, order_type, int64(target.internalId()), "stance")
		risq.projectDamage(u, target)
	}
}

func (r *GameRisq) executeSetDefaultUnitBehavior(player_id int, behavior UnitBehaviorFromFrontend, kind string) {
	player := r.players[player_id]
	if behavior.Stance != nil {
		player.default_unit_stance = defs.UnitStance(*behavior.Stance)
	}
	if behavior.Attack_back != nil {
		player.default_unit_attack_back = *behavior.Attack_back
	}
	if behavior.Interrupt_current != nil {
		player.default_unit_interrupt_current = *behavior.Interrupt_current
	}
	if behavior.Target_priority != nil {
		player.default_unit_target_priority = make([]defs.TargetCategory, 0, len(*behavior.Target_priority))
		for _, raw := range *behavior.Target_priority {
			category := defs.TargetCategory(raw)
			if category > defs.TargetCategory_NONE && category < defs.TargetCategory_END {
				player.default_unit_target_priority = append(player.default_unit_target_priority, category)
			}
		}
	}
	r.sendSetting(player.player, strings.TrimPrefix(kind, "set-")+"-set", gin.H{
		"stance":            player.default_unit_stance,
		"attack_back":       player.default_unit_attack_back,
		"interrupt_current": player.default_unit_interrupt_current,
		"target_priority":   targetCategoriesToInts(player.default_unit_target_priority),
	})
}
