package orders

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func validOrder(g *harness.Game, p int, typ defs.OrderType) defs.OrderFromFrontend {
	u, b, target := unitIDs(g, p, 1)[0], buildingID(g, p, 1), int64(0)
	subjects := []uint64{u}
	if typ.IsBuildingOrder() {
		subjects = []uint64{b}
	} else if typ.IsPlayerOrder() {
		subjects = nil
	}
	switch typ {
	case defs.OrderType_UnitMoveSpace, defs.OrderType_UnitAttackSpace:
		target = harness.SpaceKey(1, 0)
	case defs.OrderType_UnitMoveZone, defs.OrderType_UnitAttackZone:
		target = harness.ZoneKey(1, 0, 0, 0)
	case defs.OrderType_UnitGather:
		target = harness.ZoneKey(0, 0, 1, -1)
	case defs.OrderType_UnitBuild:
		target = harness.BuildKey(2, 1, 0, 0, 0)
	case defs.OrderType_UnitRepair, defs.OrderType_UnitGarrison:
		target = int64(b)
	case defs.OrderType_UnitRenew:
		target = int64(buildingID(g, p, 3))
	case defs.OrderType_UnitAttackUnit, defs.OrderType_BuildingAttackUnit:
		target = int64(unitIDs(g, g.Human(1), 1)[0])
	case defs.OrderType_UnitAttackBuilding, defs.OrderType_BuildingAttackBuilding:
		target = int64(buildingID(g, g.Human(1), 1))
	case defs.OrderType_BuildingCreate, defs.OrderType_BuildingResearch:
		target = 1
	case defs.OrderType_BuyMercenary:
		target = harness.BuildKey(11, 0, 0, 0, 0)
	case defs.OrderType_CancelOrder:
		g.Submit(p, harness.OrderProduce([]uint64{buildingID(g, p, 22)}, 12))
		g.EndTurn()
		target = int64(g.Self(p).ActiveOrders[0].InternalID)
	case defs.OrderType_CancelFoundation:
		g.Submit(p, harness.OrderBuild([]uint64{u}, 2, 3, 0, 0, 0))
		g.EndTurn()
		target = harness.ZoneKey(3, 0, 0, 0)
	}
	return harness.Order(typ, subjects, target, false)
}
