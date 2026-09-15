package risq

type RisqGatherPointLocationKind uint8

const (
	RisqGatherPointLocationKind_NONE RisqGatherPointLocationKind = iota
	RisqGatherPointLocationKind_SPACE
	RisqGatherPointLocationKind_ZONE
)

type RisqGatherObjectType uint8

const (
	RisqGatherObjectType_NONE RisqGatherObjectType = iota
	RisqGatherObjectType_UNIT
	RisqGatherObjectType_BUILDING
	RisqGatherObjectType_RESOURCE
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
	if gp.object_type == RisqGatherObjectType_NONE || gp.location_kind != RisqGatherPointLocationKind_ZONE {
		return nil
	}
	switch gp.object_type {
	case RisqGatherObjectType_RESOURCE:
		return []gatherPointCandidate{{OrderType_UnitGather, int64(gp.location_id)}}
	case RisqGatherObjectType_BUILDING:
		id := int64(gp.object_id)
		return []gatherPointCandidate{{OrderType_UnitGarrison, id}, {OrderType_UnitRepair, id}, {OrderType_UnitAttackBuilding, id}}
	case RisqGatherObjectType_UNIT:
		return []gatherPointCandidate{{OrderType_UnitAttackUnit, int64(gp.object_id)}}
	default:
		return nil
	}
}

func (gp *RisqGatherPoint) resolveOrder(risq *GameRisq, b *RisqBuilding, unit *RisqUnit) *RisqOrder {
	order_type, target_id := OrderType_UnitMoveZone, int64(gp.location_id)
	if gp.location_kind == RisqGatherPointLocationKind_SPACE {
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
