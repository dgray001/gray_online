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

type gatherPointCandidate struct {
	order_type OrderType
	target_id  int64
}

// Order types worth trying, in priority order; legality of each is left entirely to orderReceivable.
func (gp *RisqGatherPoint) candidates() []gatherPointCandidate {
	if gp.object_type == GatherObject_None || gp.location_kind != GatherLocation_Zone {
		return nil
	}
	switch gp.object_type {
	case GatherObject_Resource:
		return []gatherPointCandidate{{OrderType_UnitGather, int64(gp.location_id)}}
	case GatherObject_Building:
		id := int64(gp.object_id)
		return []gatherPointCandidate{{OrderType_UnitGarrison, id}, {OrderType_UnitRepair, id}, {OrderType_UnitAttackBuilding, id}}
	case GatherObject_Unit:
		return []gatherPointCandidate{{OrderType_UnitAttackUnit, int64(gp.object_id)}}
	default:
		return nil
	}
}

func (gp *RisqGatherPoint) resolveOrder(risq *GameRisq, b *RisqBuilding, unit *RisqUnit) *RisqOrder {
	order_type, target_id := OrderType_UnitMoveZone, int64(gp.location_id)
	if gp.location_kind == GatherLocation_Space {
		order_type = OrderType_UnitMoveSpace
	}
	for _, c := range gp.candidates() {
		candidate := createRisqOrder(0, c.order_type, b.player_id, map[uint64]Orderable{unit.internal_id: unit}, c.target_id, false)
		if unit.orderReceivable(candidate, risq) {
			order_type, target_id = c.order_type, c.target_id
			break
		}
	}
	return createRisqOrder(risq.nextOrderInternalId(), order_type, b.player_id, map[uint64]Orderable{unit.internal_id: unit}, target_id, false)
}
