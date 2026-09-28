package defs

type OrderableType uint8

const (
	OrderableType_NONE OrderableType = iota
	OrderableType_BUILDING
	OrderableType_UNIT
)

type OrderType uint8

const (
	OrderType_None OrderType = iota
	// A move only command where unit will not automatically attack, gather, etc
	OrderType_UnitMoveSpace
	OrderType_UnitMoveZone
	// These orders will first move the unit if necessary
	OrderType_UnitGather
	OrderType_UnitBuild
	OrderType_UnitRepair
	OrderType_UnitRenew
	OrderType_UnitAttackSpace
	OrderType_UnitAttackZone
	OrderType_UnitAttackUnit
	OrderType_UnitAttackBuilding
	OrderType_UnitAutoAttackUnit     // Server synthesized; not submitted by player
	OrderType_UnitAutoAttackBuilding // Server synthesized; not submitted by player
	OrderType_UnitGarrison
	OrderType_UnitUngarrison
	OrderType_UnitDelete
	OrderType_BuildingCreate
	OrderType_BuildingResearch
	OrderType_BuildingDelete
	OrderType_BuildingAttackUnit
	OrderType_BuildingAttackBuilding
	OrderType_BuildingAutoAttackUnit     // Server synthesized; not submitted by player
	OrderType_BuildingAutoAttackBuilding // Server synthesized; not submitted by player
	// Player-level orders with no subjects
	OrderType_CancelOrder
	OrderType_CancelFoundation
	OrderType_BuyMercenary
	OrderType_END
)

func (ot OrderType) IsClearImmune() bool {
	return ot == OrderType_BuildingCreate || ot == OrderType_BuildingResearch
}

func (ot OrderType) IsPlayerOrder() bool {
	return ot >= OrderType_CancelOrder && ot <= OrderType_BuyMercenary
}

func (ot OrderType) IsAttackOrder() bool {
	switch ot {
	case OrderType_UnitAttackSpace, OrderType_UnitAttackZone, OrderType_UnitAttackUnit, OrderType_UnitAttackBuilding,
		OrderType_UnitAutoAttackUnit, OrderType_UnitAutoAttackBuilding,
		OrderType_BuildingAttackUnit, OrderType_BuildingAttackBuilding,
		OrderType_BuildingAutoAttackUnit, OrderType_BuildingAutoAttackBuilding:
		return true
	default:
		return false
	}
}

func (ot OrderType) IsUnitOrder() bool {
	return ot >= OrderType_UnitMoveSpace && ot <= OrderType_UnitDelete
}

func (ot OrderType) IsBuildingOrder() bool {
	return ot >= OrderType_BuildingCreate && ot <= OrderType_BuildingAutoAttackBuilding
}

func (ot OrderType) IsAutoSynthesized() bool {
	switch ot {
	case OrderType_UnitAutoAttackUnit, OrderType_UnitAutoAttackBuilding,
		OrderType_BuildingAutoAttackUnit, OrderType_BuildingAutoAttackBuilding:
		return true
	default:
		return false
	}
}

func (ot OrderType) IsRefundOrder() bool {
	return ot == OrderType_CancelOrder || ot == OrderType_CancelFoundation
}

type OrderFromFrontend struct {
	Player_id             int      `json:"player_id"`
	Subjects              []uint64 `json:"subjects"`
	Order_type            uint8    `json:"order_type"`
	Target_id             int64    `json:"target_id"`
	Clear_previous_orders bool     `json:"clear_previous_orders"`
}
