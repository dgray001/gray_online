package risq

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

const aggressiveSightRadius = 2

func (u *RisqUnit) stanceReachable(target_zone *RisqZone) bool {
	if target_zone == nil || u.zone == nil {
		return false
	}
	if u.inAttackRange(target_zone) {
		return true
	}
	switch u.stance {
	case defs.UnitStance_AGGRESSIVE:
		return u.zone.space.distanceTo(target_zone.space) <= aggressiveSightRadius
	case defs.UnitStance_DEFENSIVE:
		return target_zone.space == u.zone.space
	case defs.UnitStance_PASSIVE, defs.UnitStance_STAND_GROUND:
		return false
	default:
		return false
	}
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
	if attacker_zone.space == u.zone.space {
		return zoneDistanceWithinSpace(u.zone, attacker_zone)
	}
	return int(u.zone.space.distanceTo(attacker_zone.space)) * 6
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

func (u *RisqUnit) replaceOrder(risq *GameRisq, order_type defs.OrderType, target_id int64) {
	if len(u.order_queue.active_orders) > 0 {
		current := u.order_queue.active_orders[0]
		if current.order_type == order_type && current.target_id == target_id {
			return
		}
	}
	order := createRisqOrder(risq.nextOrderInternalId(), order_type, u.player_id, map[uint64]Orderable{u.internal_id: u}, target_id, true)
	order.tick_source = "stance"
	if !u.orderReceivable(order, risq) {
		return
	}
	risq.addSyntheticOrder(order, risq.players[u.player_id], false)
}

func (u *RisqUnit) resolveStance(risq *GameRisq) {
	if u.zone == nil || u.cs.attack_type == defs.AttackType_NONE {
		return
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
		if idle_enough {
			order_type := attackOrderType(target, defs.OrderType_UnitAutoAttackUnit, defs.OrderType_UnitAutoAttackBuilding)
			u.replaceOrder(risq, order_type, int64(target.internalId()))
		}
		return
	}
	if u.stance == defs.UnitStance_PASSIVE {
		return
	}
	if (!u.interrupt_current && len(u.order_queue.active_orders) > 0) || u.holdsAssignedAttack() {
		return
	}
	switch u.stance {
	case defs.UnitStance_AGGRESSIVE:
		if target := nearbyAttackTarget(u.zone, u.player_id, u.target_priority, risq, aggressiveSightRadius); target != nil {
			order_type := attackOrderType(target, defs.OrderType_UnitAutoAttackUnit, defs.OrderType_UnitAutoAttackBuilding)
			u.replaceOrder(risq, order_type, int64(target.internalId()))
		}
	case defs.UnitStance_DEFENSIVE:
		if space_range, ranged := u.attack_range.SpaceRadius(); ranged {
			if target := nearbyInRangeTarget(u.zone, u.player_id, u.target_priority, u.inAttackRange, risq, space_range); target != nil {
				order_type := attackOrderType(target, defs.OrderType_UnitAutoAttackUnit, defs.OrderType_UnitAutoAttackBuilding)
				u.replaceOrder(risq, order_type, int64(target.internalId()))
			}
		} else if target := spaceAttackTarget(u, u.zone.space); target != nil {
			order_type := attackOrderType(target, defs.OrderType_UnitAutoAttackUnit, defs.OrderType_UnitAutoAttackBuilding)
			u.replaceOrder(risq, order_type, int64(target.internalId()))
		}
	case defs.UnitStance_STAND_GROUND:
		if space_range, ranged := u.attack_range.SpaceRadius(); ranged {
			if target := nearbyInRangeTarget(u.zone, u.player_id, u.target_priority, u.inAttackRange, risq, space_range); target != nil {
				order_type := attackOrderType(target, defs.OrderType_UnitAutoAttackUnit, defs.OrderType_UnitAutoAttackBuilding)
				u.replaceOrder(risq, order_type, int64(target.internalId()))
			}
		} else if target := zoneAttackTarget(u.zone, u.player_id, u.target_priority); target != nil {
			order_type := attackOrderType(target, defs.OrderType_UnitAutoAttackUnit, defs.OrderType_UnitAutoAttackBuilding)
			u.replaceOrder(risq, order_type, int64(target.internalId()))
		}
	}
}
