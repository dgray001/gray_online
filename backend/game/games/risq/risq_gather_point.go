package risq

type RisqGatherPointLocationKind uint8

const (
	GatherLocation_Space RisqGatherPointLocationKind = iota
	GatherLocation_Zone
)

type RisqGatherObjectType uint8

const (
	GatherObject_None RisqGatherObjectType = iota
	GatherObject_Unit
	GatherObject_Building
	GatherObject_Resource
)

type RisqGatherPoint struct {
	location_kind RisqGatherPointLocationKind
	location_id   uint64
	object_type   RisqGatherObjectType
	object_id     uint64
}

func (gp *RisqGatherPoint) resolveOrder(risq *GameRisq, b *RisqBuilding, unit *RisqUnit) *RisqOrder {
	var fallback_type OrderType
	switch gp.location_kind {
	case GatherLocation_Space:
		fallback_type = OrderType_UnitMoveSpace
	default:
		fallback_type = OrderType_UnitMoveZone
	}
	order_type, target_id := fallback_type, int64(gp.location_id)
	if gp.object_type != GatherObject_None && gp.location_kind == GatherLocation_Zone {
		if _, zone := invertZoneKey(uint(gp.location_id), risq); zone != nil && zone.space != nil {
			visibility := zone.space.getVisibility(b.player_id)
			switch gp.object_type {
			case GatherObject_Resource:
				if visibility >= VisibilityPoor && zone.resource != nil && zone.resource.internal_id == gp.object_id &&
					zone.resource.resources_left > 0 && isEconomicUnit(unit.unit_id) {
					order_type, target_id = OrderType_UnitGather, int64(zone.coordinate_key)
				}
			case GatherObject_Building:
				if target := risq.buildings[gp.object_id]; target != nil && !target.isDeleted() && visibility >= VisibilityPoor {
					if target == b && !target.underConstruction() && uint16(len(target.garrisoned_units)) < target.garrison_capacity {
						order_type, target_id = OrderType_UnitGarrison, int64(target.internal_id)
					} else if target.player_id == b.player_id && isEconomicUnit(unit.unit_id) && !target.underConstruction() &&
						target.cs.health < float64(target.cs.max_health) {
						order_type, target_id = OrderType_UnitRepair, int64(target.internal_id)
					} else if canAttack(b.player_id, target.player_id) && !isEconomicUnit(unit.unit_id) {
						order_type, target_id = OrderType_UnitAttackBuilding, int64(target.internal_id)
					}
				}
			case GatherObject_Unit:
				if target := risq.units[gp.object_id]; target != nil && !target.isDeleted() && visibility >= VisibilityGood &&
					canAttack(b.player_id, target.player_id) && !isEconomicUnit(unit.unit_id) {
					order_type, target_id = OrderType_UnitAttackUnit, int64(target.internal_id)
				}
			}
		}
	}
	return createRisqOrder(risq.nextOrderInternalId(), order_type, b.player_id, map[uint64]Orderable{unit.internal_id: unit}, target_id, false)
}
