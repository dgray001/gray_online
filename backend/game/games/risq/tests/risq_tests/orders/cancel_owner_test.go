package orders

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestCancelCannotTargetAnotherPlayersOrder(t *testing.T) {
	g := orderGame(t, richBank, "")
	p, other := g.Human(0), g.Human(1)
	g.Submit(p, harness.OrderMove(unitIDs(g, p, 1)[:1], 3, 0))
	g.EndTurn()
	id := g.Self(p).ActiveOrders[0].InternalID
	g.SubmitRejected(other, harness.Order(defs.OrderType_CancelOrder, nil, int64(id), false), fmt.Sprintf("No active order with id %d", id))
	if g.Self(p).ActiveOrders[0].InternalID != id {
		t.Fatal("foreign cancellation changed owner's queue")
	}
}
