package risq

import "github.com/dgray001/gray_online/game/games/risq/internal/defs"

func metricAttackTarget(a *metricActor, r *GameRisq) *RisqSpace {
	orders := a.base.order_queue.active_orders
	if len(orders) == 0 || !orders[0].order_type.IsAttackOrder() {
		return nil
	}
	o := orders[0]
	switch o.order_type {
	case defs.OrderType_UnitAttackSpace:
		return invertSpaceKey(uint(o.target_id), r)
	case defs.OrderType_UnitAttackZone:
		_, zone := invertZoneKey(uint(o.target_id), r)
		return zone.space
	case defs.OrderType_UnitAttackUnit, defs.OrderType_UnitAutoAttackUnit:
		if target := r.units[uint64(o.target_id)]; target != nil && target.zone != nil {
			return target.zone.space
		}
	case defs.OrderType_UnitAttackBuilding, defs.OrderType_UnitAutoAttackBuilding:
		if target := r.buildings[uint64(o.target_id)]; target != nil && target.zone != nil {
			return target.zone.space
		}
	}
	return nil
}
