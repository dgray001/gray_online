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

func (v *aiView) AssignedTo(building bool, internal_id uint64, exclude map[uint64]bool) int {
	v.initQueues()
	assigned := map[uint64]bool{}
	for subject, queue := range v.queues {
		if subject >= 1_000_000_000 || exclude[subject] {
			continue
		}
		for _, raw := range queue {
			if target, ok := attackOrderTarget(raw); ok && target == (attackTarget{building: building, id: internal_id}) {
				assigned[subject] = true
			}
		}
	}
	return len(assigned)
}
