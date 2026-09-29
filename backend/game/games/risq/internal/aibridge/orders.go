package aibridge

import (
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
)

func unitOrder(u ai.UnitView, order_type defs.OrderType, target_id int64, clear_previous bool) ai.Order {
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: uint8(order_type), TargetID: target_id, ClearPreviousOrders: clear_previous}
}

func buildingOrder(b ai.BuildingView, order_type defs.OrderType, target_id int64) ai.Order {
	return ai.Order{Subjects: []uint64{b.InternalID}, OrderType: uint8(order_type), TargetID: target_id}
}

func (v *aiView) MoveOrder(u ai.UnitView, target ai.ZoneRef, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindMove, TargetZone: &target})
	return unitOrder(u, defs.OrderType_UnitMoveZone, int64(zoneKey(target)), clear_previous)
}

func (v *aiView) GatherOrder(u ai.UnitView, target ai.ResourceView, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindGather, TargetResource: &target})
	return unitOrder(u, defs.OrderType_UnitGather, int64(zoneKey(target.Location)), clear_previous)
}

func (v *aiView) BuildOrder(u ai.UnitView, building_id uint32, target ai.ZoneRef, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindBuild, TargetZone: &target})
	if !v.occupied(target) && !v.planned[target] {
		v.ordered_foundations[target] = building_id
	}
	return unitOrder(u, defs.OrderType_UnitBuild, int64(util.Pair(int(building_id), int(zoneKey(target)))), clear_previous)
}

func (v *aiView) RepairOrder(u ai.UnitView, target ai.BuildingView, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindRepair, TargetBuilding: &target})
	return unitOrder(u, defs.OrderType_UnitRepair, int64(target.InternalID), clear_previous)
}

func (v *aiView) RenewOrder(u ai.UnitView, target ai.BuildingView, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindRenew, TargetBuilding: &target})
	return unitOrder(u, defs.OrderType_UnitRenew, int64(target.InternalID), clear_previous)
}

func (v *aiView) AttackUnitOrder(u ai.UnitView, target ai.UnitView, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindAttackUnit, TargetUnit: &target})
	return unitOrder(u, defs.OrderType_UnitAttackUnit, int64(target.InternalID), clear_previous)
}

func (v *aiView) AttackBuildingOrder(u ai.UnitView, target ai.BuildingView, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindAttackBuilding, TargetBuilding: &target})
	return unitOrder(u, defs.OrderType_UnitAttackBuilding, int64(target.InternalID), clear_previous)
}

func (v *aiView) AttackSpaceOrder(u ai.UnitView, target ai.Coordinate, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindAttackSpace, TargetSpace: &target})
	return unitOrder(u, defs.OrderType_UnitAttackSpace, int64(spaceKey(target)), clear_previous)
}

func (v *aiView) AttackZoneOrder(u ai.UnitView, target ai.ZoneRef, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindAttackZone, TargetZone: &target})
	return unitOrder(u, defs.OrderType_UnitAttackZone, int64(zoneKey(target)), clear_previous)
}

func (v *aiView) GarrisonOrder(u ai.UnitView, target ai.BuildingView, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindGarrison, TargetBuilding: &target})
	return unitOrder(u, defs.OrderType_UnitGarrison, int64(target.InternalID), clear_previous)
}

func (v *aiView) UngarrisonOrder(u ai.UnitView, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindUngarrison})
	return unitOrder(u, defs.OrderType_UnitUngarrison, 0, clear_previous)
}

func (v *aiView) DeleteUnitOrder(u ai.UnitView) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindDelete})
	return unitOrder(u, defs.OrderType_UnitDelete, 0, true)
}

func (v *aiView) CreateUnitOrder(b ai.BuildingView, unit_id uint32) ai.Order {
	v.claimBuilding(b.InternalID)
	v.planned_production[b.InternalID]++
	return buildingOrder(b, defs.OrderType_BuildingCreate, int64(unit_id))
}

func (v *aiView) ResearchOrder(b ai.BuildingView, tech_id uint32) ai.Order {
	v.claimBuilding(b.InternalID)
	v.planned_production[b.InternalID]++
	v.planned_techs[tech_id] = true
	return buildingOrder(b, defs.OrderType_BuildingResearch, int64(tech_id))
}

func (v *aiView) DeleteBuildingOrder(b ai.BuildingView) ai.Order {
	v.claimBuilding(b.InternalID)
	return buildingOrder(b, defs.OrderType_BuildingDelete, 0)
}

func (v *aiView) BuildingAttackUnitOrder(b ai.BuildingView, target ai.UnitView) ai.Order {
	v.claimBuilding(b.InternalID)
	return buildingOrder(b, defs.OrderType_BuildingAttackUnit, int64(target.InternalID))
}

func (v *aiView) BuildingAttackBuildingOrder(b ai.BuildingView, target ai.BuildingView) ai.Order {
	v.claimBuilding(b.InternalID)
	return buildingOrder(b, defs.OrderType_BuildingAttackBuilding, int64(target.InternalID))
}

func (v *aiView) CancelOrder(order_id uint64) ai.Order {
	v.cancelled_orders[order_id] = true
	v.gather_counts = nil
	return ai.Order{OrderType: uint8(defs.OrderType_CancelOrder), TargetID: int64(order_id)}
}

func (v *aiView) CancelFoundationOrder(f ai.FoundationView) ai.Order {
	v.cancelled_foundations[f.Location] = true
	return ai.Order{OrderType: uint8(defs.OrderType_CancelFoundation), TargetID: int64(zoneKey(f.Location))}
}

func (v *aiView) HireMercenaryOrder(unit_id uint32, target ai.ZoneRef) ai.Order {
	return ai.Order{OrderType: uint8(defs.OrderType_BuyMercenary), TargetID: int64(util.Pair(int(unit_id), int(zoneKey(target))))}
}
