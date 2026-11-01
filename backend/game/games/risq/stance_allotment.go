package risq

import (
	"cmp"
	"slices"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

func (u *RisqUnit) stanceTarget(r *GameRisq) Attackable {
	active := u.order_queue.active_orders
	if len(active) == 0 || active[0].tick_source != "stance" {
		return nil
	}
	target := r.attackOrderTarget(active[0])
	if target == nil || !target.isAlive() || target.isDeleted() {
		return nil
	}
	return target
}

func (r *GameRisq) prepareStancePicks(orderables []Orderable) {
	r.stance_picks = nil
	for _, o := range orderables {
		unit, ok := o.(*RisqUnit)
		if !ok {
			continue
		}
		if len(unit.order_queue.active_orders) > 0 {
			order_type := unit.order_queue.active_orders[0].order_type
			if order_type == defs.OrderType_UnitAutoAttackUnit || order_type == defs.OrderType_UnitAutoAttackBuilding {
				unit.order_queue.nextOrder(unit, r)
			}
		}
		unit.resolveStance(r)
	}
}

func stackRank(unit *RisqUnit) int {
	if unit.attack_stack == 0 {
		return 0
	}
	return 256 - int(unit.attack_stack)
}

func (r *GameRisq) computeStanceAllotments(orderables []Orderable) {
	picks := r.stance_picks
	r.stance_picks = nil
	reassigned := make([]stancePick, 0, len(picks))
	moving := make(map[*RisqUnit]bool)
	for _, pick := range picks {
		if !slices.Contains(pick.candidates, pick.unit.stanceTarget(r)) {
			reassigned = append(reassigned, pick)
			moving[pick.unit] = true
		}
	}
	counts := make(map[Attackable]int)
	for _, o := range orderables {
		unit, ok := o.(*RisqUnit)
		if !ok || moving[unit] {
			continue
		}
		if target := unit.stanceTarget(r); target != nil {
			counts[target]++
		}
	}
	slices.SortFunc(reassigned, func(a, b stancePick) int {
		return cmp.Or(cmp.Compare(stackRank(a.unit), stackRank(b.unit)), cmp.Compare(a.unit.internal_id, b.unit.internal_id))
	})
	for _, pick := range reassigned {
		target := chooseStanceTarget(pick, counts)
		counts[target]++
		order_type := attackOrderType(target, defs.OrderType_UnitAutoAttackUnit, defs.OrderType_UnitAutoAttackBuilding)
		pick.unit.replaceOrder(r, order_type, int64(target.internalId()), "stance")
	}
}
