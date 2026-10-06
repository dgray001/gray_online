package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestMoveCarriesAcrossTurnsWithStableOrderID(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	id := unitIDs(g, p, 1)[0]
	g.Submit(p, harness.OrderMove([]uint64{id}, 3, 0))
	if len(g.State(p).Unit(id).ActiveOrders) != 0 {
		t.Fatal("order delivered before all players submitted")
	}
	g.EndTurn()
	state := g.Self(p)
	if len(state.ActiveOrders) != 1 || state.ActiveOrders[0].InternalID == 0 {
		t.Fatalf("missing received order: %+v", state.ActiveOrders)
	}
	orderID := state.ActiveOrders[0].InternalID
	if g.State(p).Unit(id).Space == (harness.Coord{X: 3}) {
		t.Fatal("move unexpectedly completed in one turn")
	}
	g.EndTurn()
	state = g.Self(p)
	if len(state.ActiveOrders) != 1 || state.ActiveOrders[0].InternalID != orderID {
		t.Fatalf("carried order changed: %+v", state.ActiveOrders)
	}
	for turn := 0; turn < 4 && len(g.Self(p).ActiveOrders) > 0; turn++ {
		g.EndTurn()
	}
	u := g.State(p).Unit(id)
	if u.Space != (harness.Coord{X: 3}) || len(u.ActiveOrders) != 0 || len(g.Self(p).ActiveOrders) != 0 || g.Self(p).OrdersSubmitted {
		t.Fatalf("move did not finish/reset: %+v", u)
	}
}
