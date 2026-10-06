package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestUnsubmitDiscardsOnlyNewOrders(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	id := unitIDs(g, p, 1)[0]
	g.Submit(p, harness.OrderMove([]uint64{id}, 3, 0))
	g.EndTurn()
	old := g.Self(p).ActiveOrders[0].InternalID
	g.Submit(p, harness.OrderMoveZone([]uint64{id}, 0, 0, 1, 0))
	if len(g.Self(p).ActiveOrders) != 2 {
		t.Fatal("new order was not appended")
	}
	g.Action(p, "unsubmit-orders", nil)
	state := g.Self(p)
	if state.OrdersSubmitted || len(state.ActiveOrders) != 1 || state.ActiveOrders[0].InternalID != old {
		t.Fatalf("unsubmit lost received order: %+v", state)
	}
	g.Submit(p)
	g.EndTurn()
	if g.State(p).Unit(id).Space.X == 0 {
		t.Fatal("unsubmitted replacement still executed")
	}
}

func TestUnsubmitAllowsReplacingPendingOrdersWithoutSpending(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	g.Submit(p, harness.OrderProduce([]uint64{buildingID(g, p, 1)}, 1))
	if g.Self(p).Resources.Food != 1000 {
		t.Fatal("pending production charged before receipt")
	}
	g.Action(p, "unsubmit-orders", nil)
	if len(g.Self(p).ActiveOrders) != 0 {
		t.Fatal("pending production survived unsubmit")
	}
	g.Submit(p)
	g.EndTurn()
	if len(g.Self(p).Units) != 3 || g.Self(p).Resources.Food != 1000 {
		t.Fatal("discarded production executed or spent")
	}
}
