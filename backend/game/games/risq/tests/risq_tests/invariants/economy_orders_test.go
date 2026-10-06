package invariants

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func economyOrders(g *harness.Game, turn, slot int) []defs.OrderFromFrontend {
	ids, home := unitIDs(g, slot, 1), homes[slot]
	center := buildingID(g, slot, 1)
	switch turn {
	case 0:
		return []defs.OrderFromFrontend{
			harness.OrderGather(ids[:1], home[0], home[1], 1, 0),
			harness.OrderBuild(ids[1:2], 2, home[0], home[1], 0, -1),
			harness.OrderGarrison(ids[2:3], center), harness.OrderProduce([]uint64{center}, 1),
		}
	case 1:
		return []defs.OrderFromFrontend{harness.OrderProduce([]uint64{center}, 1)}
	case 2:
		return []defs.OrderFromFrontend{harness.Order(defs.OrderType_UnitUngarrison, ids[2:3], 0, false), harness.Order(defs.OrderType_BuildingResearch, []uint64{center}, 1, false)}
	case 3:
		return []defs.OrderFromFrontend{harness.Order(defs.OrderType_UnitDelete, ids[len(ids)-1:], 0, false)}
	case 4:
		return []defs.OrderFromFrontend{harness.Order(defs.OrderType_BuildingDelete, []uint64{buildingID(g, slot, 2)}, 0, false)}
	case 5:
		return []defs.OrderFromFrontend{harness.Order(defs.OrderType_UnitMoveSpace, ids, harness.SpaceKey(0, 0), true)}
	}
	return nil
}
