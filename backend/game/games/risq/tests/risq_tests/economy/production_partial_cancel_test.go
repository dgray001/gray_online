package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestPartiallyProducedUnitCancellationRefundsOnce(t *testing.T) {
	g := economyGame(t, `{"food":100,"wood":100}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":22,"player":0},"units":[{"id":1,"player":0,"count":1}]}]},`+remoteVillager, `,"unlimited_population":true`)
	p := g.Human(0)
	g.Submit(p, harness.OrderProduce([]uint64{g.Self(p).Buildings[0].InternalID}, 12))
	g.EndTurn()
	state := g.Self(p)
	if len(state.ActiveOrders) != 1 || len(state.Buildings[0].ProductionQueue) != 1 || state.Buildings[0].ProductionQueue[0].StaminaRemaining != 4 || state.Resources.Food != 30 || state.Resources.Wood != 60 {
		t.Fatalf("production did not make partial progress: %+v", state)
	}
	g.Submit(p, harness.Order(defs.OrderType_CancelOrder, nil, int64(state.ActiveOrders[0].InternalID), false))
	for turn := 0; turn < 3; turn++ {
		g.EndTurn()
		state = g.Self(p)
		if state.Resources.Food != 100 || state.Resources.Wood != 100 || len(state.Units) != 1 || len(state.ActiveOrders) != 0 || len(state.Buildings[0].ProductionQueue) != 0 {
			t.Errorf("partial cancellation retained payment or spawned a unit: %+v", state)
		}
	}
}
