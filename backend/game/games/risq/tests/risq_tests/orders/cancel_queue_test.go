package orders

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestCancelQueuedOrderPreservesFrontOrder(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	id := unitIDs(g, p, 1)[0]
	g.Submit(p, harness.OrderMove([]uint64{id}, 3, 0), harness.OrderMove([]uint64{id}, 0, 0))
	g.EndTurn()
	queue := g.State(p).Unit(id).ActiveOrders
	if len(queue) != 2 {
		t.Fatalf("setup queue: %v", queue)
	}
	g.Submit(p, harness.Order(defs.OrderType_CancelOrder, nil, int64(queue[1].InternalID), false))
	g.EndTurn()
	if got := g.State(p).Unit(id).ActiveOrders; len(got) != 1 || got[0].InternalID != queue[0].InternalID {
		t.Fatalf("wrong order cancelled: %v", got)
	}
	for range 5 {
		g.EndTurn()
	}
	if g.State(p).Unit(id).Space != (harness.Coord{X: 3}) {
		t.Fatal("cancelled return move executed")
	}
	g.SubmitRejected(p, harness.Order(defs.OrderType_CancelOrder, nil, int64(queue[1].InternalID), false), "No active order with id "+fmt.Sprint(queue[1].InternalID))
}
