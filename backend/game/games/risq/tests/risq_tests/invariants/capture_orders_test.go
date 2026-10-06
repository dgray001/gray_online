package invariants

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func captureOrders(g *harness.Game, turn, slot int) []defs.OrderFromFrontend {
	if slot == 1 && turn == 2 {
		return []defs.OrderFromFrontend{harness.Order(defs.OrderType_BuildingDelete, []uint64{buildingID(g, 1, 1)}, 0, false)}
	}
	if slot == 0 && turn != 2 {
		x := 0
		if turn == 1 {
			x = 1
		}
		return []defs.OrderFromFrontend{harness.OrderMove(unitIDs(g, 0, 11), x, 0)}
	}
	return nil
}
