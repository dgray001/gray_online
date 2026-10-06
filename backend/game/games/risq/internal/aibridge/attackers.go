package aibridge

import "github.com/dgray001/gray_online/game/games/risq/internal/defs"

// The target of an attack-unit or attack-building order, whether manual or engine-chosen
func attackOrderTarget(o snapOrder) (attackTarget, bool) {
	switch o.OrderType {
	case defs.OrderType_UnitAttackUnit, defs.OrderType_UnitAutoAttackUnit:
		return attackTarget{id: uint64(o.TargetId)}, true
	case defs.OrderType_UnitAttackBuilding, defs.OrderType_UnitAutoAttackBuilding:
		return attackTarget{building: true, id: uint64(o.TargetId)}, true
	}
	return attackTarget{}, false
}

func (v *aiView) addAttacker(target attackTarget, unit_id uint64) {
	if v.attackers[target] == nil {
		v.attackers[target] = map[uint64]bool{}
	}
	v.attackers[target][unit_id] = true
}

// A clearing order drops the unit from every target it was attacking before
func (v *aiView) recordAttacker(target attackTarget, unit_id uint64, clear_previous bool) {
	if clear_previous {
		for _, units := range v.attackers {
			delete(units, unit_id)
		}
	}
	v.addAttacker(target, unit_id)
}

func (v *aiView) AssignedTo(building bool, internal_id uint64, exclude map[uint64]bool) int {
	count := 0
	for unit_id := range v.attackers[attackTarget{building: building, id: internal_id}] {
		if !exclude[unit_id] {
			count++
		}
	}
	return count
}

// A cancelled attack order stops counting its unit against the target, unless another live order of the unit still hits it
func (v *aiView) dropCancelledAttacker(order_id uint64) {
	for i := range v.me.Units {
		unit := &v.me.Units[i]
		for _, o := range unit.ActiveOrders {
			target, ok := attackOrderTarget(o)
			if !ok || o.InternalId != order_id {
				continue
			}
			for _, other := range v.liveOrders(unit.ActiveOrders) {
				if other_target, is_attack := attackOrderTarget(other); is_attack && other_target == target {
					return
				}
			}
			delete(v.attackers[target], unit.InternalId)
			return
		}
	}
}

func (v *aiView) loadAttackers() {
	for i := range v.me.Units {
		for _, o := range v.me.Units[i].ActiveOrders {
			if target, ok := attackOrderTarget(o); ok {
				v.addAttacker(target, v.me.Units[i].InternalId)
			}
		}
	}
}
