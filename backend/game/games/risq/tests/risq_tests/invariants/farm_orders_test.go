package invariants

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func farmOrders(g *harness.Game, turn, slot int) []defs.OrderFromFrontend {
	if turn%2 == 1 {
		return []defs.OrderFromFrontend{harness.Order(defs.OrderType_UnitRenew, unitIDs(g, slot, 1), int64(buildingID(g, slot, 3)), false)}
	}
	home := homes[slot]
	return []defs.OrderFromFrontend{harness.OrderGather(unitIDs(g, slot, 1), home[0], home[1], 1, 0)}
}

func verifyFarm(g *harness.Game, turn int) {
	g.T.Helper()
	for slot := range 2 {
		state := g.Self(g.Human(slot))
		if len(state.Refusals()) != 0 || state.Resources.Food != 500+2.5*float64(turn/2+1) || state.Resources.Wood != 500-60*float64((turn+1)/2) {
			g.T.Fatalf("farm cycle %d bank %+v, failures %v", turn, state.Resources, state.Refusals())
		}
	}
}
