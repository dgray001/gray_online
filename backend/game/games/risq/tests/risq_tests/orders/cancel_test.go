package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestCancelOneSubjectLeavesOtherSubjectsMoving(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	ids := unitIDs(g, p, 1)
	g.Submit(p, harness.OrderMove(ids, 3, 0))
	g.EndTurn()
	orderID := g.Self(p).ActiveOrders[0].InternalID
	stopped := *g.State(p).Unit(ids[0])
	g.Submit(p, harness.Order(defs.OrderType_CancelOrder, ids[:1], int64(orderID), false))
	g.EndTurn()
	if u := g.State(p).Unit(ids[0]); u.Space != stopped.Space || u.Zone != stopped.Zone || len(u.ActiveOrders) != 0 {
		t.Fatalf("cancelled subject kept moving: %+v", u)
	}
	if u := g.State(p).Unit(ids[1]); len(u.ActiveOrders) != 1 || u.ActiveOrders[0].InternalID != orderID {
		t.Fatalf("other subject lost shared order: %+v", u)
	}
	g.Submit(p, harness.Order(defs.OrderType_CancelOrder, nil, int64(orderID), false))
	g.EndTurn()
	if len(g.Self(p).ActiveOrders) != 0 {
		t.Fatal("whole-order cancellation left active work")
	}
	g.EndTurn()
	if u := g.State(p).Unit(ids[0]); u.Space != stopped.Space || u.Zone != stopped.Zone {
		t.Fatal("cancelled move restarted")
	}
}
