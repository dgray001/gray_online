package risq

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
	"sort"
)

type RisqGatherPointLocationKind uint8

const (
	RisqGatherPointLocationKind_NONE RisqGatherPointLocationKind = iota
	RisqGatherPointLocationKind_SPACE
	RisqGatherPointLocationKind_ZONE
	RisqGatherPointLocationKind_END
)

type RisqGatherObjectType uint8

const (
	RisqGatherObjectType_NONE RisqGatherObjectType = iota
	RisqGatherObjectType_UNIT
	RisqGatherObjectType_BUILDING
	RisqGatherObjectType_RESOURCE
	RisqGatherObjectType_END
)

type RisqGatherPoint struct {
	location_kind RisqGatherPointLocationKind
	location_id   uint64
	object_type   RisqGatherObjectType
	object_id     uint64
}

func (r *GameRisq) applyUngarrisonGatherPoints(orderables []Orderable) {
	units := make([]*RisqUnit, 0)
	for _, orderable := range orderables {
		if u, ok := orderable.(*RisqUnit); ok {
			if _, leaving := u.intent.detail.(*UngarrisonIntent); leaving {
				units = append(units, u)
			}
		}
	}
	sort.Slice(units, func(i, j int) bool { return units[i].internal_id < units[j].internal_id })
	for _, u := range units {
		building := u.intent.detail.(*UngarrisonIntent).building
		active := u.order_queue.active_orders
		if u.deleted || u.garrisoned_in != nil || building.deleted || len(active) != 1 || active[0].order_type != defs.OrderType_UnitUngarrison {
			continue
		}
		point := building.gather_point
		if point == nil || (point.object_type == RisqGatherObjectType_BUILDING && point.object_id == building.internal_id) {
			continue
		}
		r.addSyntheticOrder(point.resolveOrder(r, building, u), r.players[u.player_id], false)
	}
}

func (gp *RisqGatherPoint) toFrontend() gin.H {
	return gin.H{
		"location_kind": gp.location_kind,
		"location_id":   gp.location_id,
		"object_type":   gp.object_type,
		"object_id":     gp.object_id,
	}
}

type gatherPointCandidate struct {
	order_type defs.OrderType
	target_id  int64
}

// Order types worth trying, in priority order; legality of each is left entirely to orderReceivable.
func (gp *RisqGatherPoint) candidates() []gatherPointCandidate {
	if gp.object_type == RisqGatherObjectType_NONE || gp.location_kind != RisqGatherPointLocationKind_ZONE {
		return nil
	}
	switch gp.object_type {
	case RisqGatherObjectType_RESOURCE:
		return []gatherPointCandidate{{defs.OrderType_UnitGather, int64(gp.location_id)}}
	case RisqGatherObjectType_BUILDING:
		if gp.object_id == 0 {
			return nil
		}
		id := int64(gp.object_id)
		return []gatherPointCandidate{{defs.OrderType_UnitGarrison, id}, {defs.OrderType_UnitRepair, id}, {defs.OrderType_UnitAttackBuilding, id}}
	case RisqGatherObjectType_UNIT:
		return []gatherPointCandidate{{defs.OrderType_UnitAttackUnit, int64(gp.object_id)}}
	default:
		return nil
	}
}

func (gp *RisqGatherPoint) foundationTarget(r *GameRisq, unit *RisqUnit) int64 {
	if unit.unitType() != defs.UnitType_ECONOMIC || gp.location_kind != RisqGatherPointLocationKind_ZONE || gp.object_type != RisqGatherObjectType_BUILDING {
		return 0
	}
	_, zone := invertZoneKey(uint(gp.location_id), r)
	if zone == nil {
		return 0
	}
	var id uint32
	if b := zone.building; b != nil && (gp.object_id == 0 || b.internal_id == gp.object_id) && !b.deleted && b.player_id == unit.player_id && b.underConstruction() {
		id = b.building_id
	} else if f := r.players[unit.player_id].planned_foundations[zone.coordinate_key]; gp.object_id == 0 && f != nil {
		id = f.building_id
	}
	if id == 0 {
		return 0
	}
	return int64(util.Pair(int(id), int(zone.coordinate_key)))
}

func (gp *RisqGatherPoint) resolveOrder(risq *GameRisq, b *RisqBuilding, unit *RisqUnit) (result *RisqOrder) {
	defer func() { result.tick_source = "gather_point" }()
	order_type, target_id := defs.OrderType_UnitMoveZone, int64(gp.location_id)
	if gp.location_kind == RisqGatherPointLocationKind_SPACE {
		order_type = defs.OrderType_UnitMoveSpace
	}
	if target := gp.foundationTarget(risq, unit); target != 0 {
		order := createRisqOrder(0, defs.OrderType_UnitBuild, b.player_id, map[uint64]Orderable{unit.internal_id: unit}, target, false)
		if unit.orderReceivable(order, risq) {
			return createRisqOrder(risq.nextOrderInternalId(), order.order_type, b.player_id, order.subjects, target, false)
		}
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
