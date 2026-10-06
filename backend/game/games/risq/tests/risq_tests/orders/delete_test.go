package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestDeleteWithClearPreviousRemovesUnitAndItsOrders(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	id := unitIDs(g, p, 1)[0]
	g.Submit(p, harness.OrderMove([]uint64{id}, 3, 0), harness.Order(defs.OrderType_UnitDelete, []uint64{id}, 0, true))
	g.EndTurn()
	if g.State(p).Unit(id) != nil || len(g.Self(p).ActiveOrders) != 0 || len(g.Self(p).Units) != 2 {
		t.Fatal("deleted unit or cleared orders remained")
	}
	g.EndTurn()
	if g.State(p).Unit(id) != nil {
		t.Fatal("deleted unit returned")
	}
}
