package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestCancelOnlyOneProducerRefundsOnlyItsReservation(t *testing.T) {
	g := orderGame(t, `{"food":140,"wood":80}`, "")
	p := g.Human(0)
	config := defs.BuildingConfigs[1]
	config.Produces = append(append([]defs.Producible(nil), config.Produces...), defs.Producible{Kind: defs.ProducibleKind_UNIT, Id: 12})
	defs.BuildingConfigs[1] = config
	center, barracks := buildingID(g, p, 1), buildingID(g, p, 22)
	g.Submit(p, harness.OrderProduce([]uint64{center, barracks}, 12))
	g.EndTurn()
	state := g.Self(p)
	if len(state.ActiveOrders) != 1 || state.Resources.Food != 0 || state.Resources.Wood != 0 {
		t.Fatal("setup did not reserve both productions")
	}
	g.Submit(p, harness.Order(defs.OrderType_CancelOrder, []uint64{barracks}, int64(state.ActiveOrders[0].InternalID), false))
	g.EndTurn()
	state = g.Self(p)
	if state.Resources.Food != 70 || state.Resources.Wood != 40 || len(unitIDs(g, p, 12)) != 1 || len(state.ActiveOrders) != 0 {
		t.Fatalf("partial production cancellation affected other subject: %+v", state)
	}
	g.EndTurn()
	if g.Self(p).Resources.Food != 70 || len(unitIDs(g, p, 12)) != 1 {
		t.Fatal("cancelled production restarted or refunded twice")
	}
}
