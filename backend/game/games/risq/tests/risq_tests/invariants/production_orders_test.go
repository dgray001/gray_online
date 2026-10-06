package invariants

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func productionOrders(g *harness.Game, turn, slot int) []defs.OrderFromFrontend {
	if turn == 2 {
		return []defs.OrderFromFrontend{harness.Order(defs.OrderType_UnitDelete, unitIDs(g, slot, 1)[:1], 0, false)}
	}
	if turn != 0 {
		return nil
	}
	var ids []uint64
	for _, building := range g.Self(g.Human(slot)).Buildings {
		ids = append(ids, building.InternalID)
	}
	return []defs.OrderFromFrontend{harness.OrderProduce(ids, 1)}
}
