package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestProductionCancellationRefundsReservedFood(t *testing.T) {
	g := productionGame(t, 5, 1, "")
	p := g.Human(0)
	g.Submit(p, harness.OrderProduce([]uint64{centers(g, p)[0].InternalID}, 1))
	g.EndTurn()
	state := g.Self(p)
	if len(state.ActiveOrders) != 1 || state.Resources.Food != 250 {
		t.Fatalf("setup: orders %v, food %v", state.ActiveOrders, state.Resources.Food)
	}
	g.Submit(p, harness.Order(defs.OrderType_CancelOrder, nil, int64(state.ActiveOrders[0].InternalID), false))
	g.EndTurn()
	for turn := 0; turn < 2; turn++ {
		state = g.Self(p)
		if state.Resources.Food != 300 || len(state.Units) != 5 || len(centers(g, p)[0].ProductionQueue) != 0 {
			t.Errorf("cancelled production: food %v, units %d, queue %v", state.Resources.Food, len(state.Units), centers(g, p)[0].ProductionQueue)
		}
		g.EndTurn()
	}
}
