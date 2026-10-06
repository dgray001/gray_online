package harness

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
)

// Keys encode a coordinate into one number exactly as the engine and frontend do: a pair of a pair

func SpaceKey(x, y int) int64 {
	return int64(util.Pair(x, y))
}

// A zone key pairs its space key with the zone's local coordinate
func ZoneKey(x, y, zx, zy int) int64 {
	return int64(util.Pair(int(SpaceKey(x, y)), int(util.Pair(zx, zy))))
}

func BuildKey(buildingId uint32, x, y, zx, zy int) int64 {
	return int64(util.Pair(int(buildingId), int(ZoneKey(x, y, zx, zy))))
}

// Game.Submit overwrites the order's player with the submitter, so builders leave it zero
func Order(typ defs.OrderType, subjects []uint64, target int64, clear bool) defs.OrderFromFrontend {
	return defs.OrderFromFrontend{Subjects: subjects, Order_type: uint8(typ), Target_id: target, Clear_previous_orders: clear}
}

func OrderMove(subjects []uint64, x, y int) defs.OrderFromFrontend {
	return Order(defs.OrderType_UnitMoveSpace, subjects, SpaceKey(x, y), false)
}

func OrderMoveZone(subjects []uint64, x, y, zx, zy int) defs.OrderFromFrontend {
	return Order(defs.OrderType_UnitMoveZone, subjects, ZoneKey(x, y, zx, zy), false)
}

func OrderGarrison(subjects []uint64, building uint64) defs.OrderFromFrontend {
	return Order(defs.OrderType_UnitGarrison, subjects, int64(building), false)
}

func OrderGather(subjects []uint64, x, y, zx, zy int) defs.OrderFromFrontend {
	return Order(defs.OrderType_UnitGather, subjects, ZoneKey(x, y, zx, zy), false)
}

func OrderAttackUnit(subjects []uint64, target uint64) defs.OrderFromFrontend {
	return Order(defs.OrderType_UnitAttackUnit, subjects, int64(target), false)
}

func OrderBuild(subjects []uint64, buildingId uint32, x, y, zx, zy int) defs.OrderFromFrontend {
	return Order(defs.OrderType_UnitBuild, subjects, BuildKey(buildingId, x, y, zx, zy), false)
}

func OrderProduce(subjects []uint64, unitId uint32) defs.OrderFromFrontend {
	return Order(defs.OrderType_BuildingCreate, subjects, int64(unitId), false)
}
