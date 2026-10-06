package invariants

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func evictionOrders(g *harness.Game, turn, slot int) []defs.OrderFromFrontend {
	if turn == 0 || turn == 2 {
		return []defs.OrderFromFrontend{harness.OrderGarrison(unitIDs(g, slot, 1), buildingID(g, slot, 1))}
	}
	if turn == 1 {
		for _, unit := range g.Self(g.Human(slot)).Units {
			if unit.GarrisonedIn != nil {
				return []defs.OrderFromFrontend{harness.Order(defs.OrderType_UnitDelete, []uint64{unit.InternalID}, 0, false)}
			}
		}
		g.T.Fatal("garrison capacity contest admitted no units")
	}
	if turn == 3 {
		return []defs.OrderFromFrontend{harness.Order(defs.OrderType_BuildingDelete, []uint64{buildingID(g, slot, 1)}, 0, false)}
	}
	return nil
}
