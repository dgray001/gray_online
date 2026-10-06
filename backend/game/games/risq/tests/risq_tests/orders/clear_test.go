package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestClearPreviousReplacesOnlySelectedSubjectsOrders(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	ids := unitIDs(g, p, 1)
	g.Submit(p, harness.OrderMove(ids, 3, 0), harness.Order(defs.OrderType_UnitDelete, ids[:1], 0, false))
	g.EndTurn()
	old := g.Self(p).ActiveOrders[0].InternalID
	replacement := harness.Order(defs.OrderType_UnitMoveSpace, ids[:1], harness.SpaceKey(0, 0), true)
	g.Submit(p, replacement)
	g.EndTurn()
	if u := g.State(p).Unit(ids[0]); u == nil || u.Space != (harness.Coord{}) || len(u.ActiveOrders) != 0 {
		t.Fatalf("replacement did not clear queue: %+v", u)
	}
	if u := g.State(p).Unit(ids[1]); len(u.ActiveOrders) != 1 || u.ActiveOrders[0].InternalID != old {
		t.Fatalf("unselected subject lost its order: %+v", u)
	}
}

func TestRejectedReplacementDoesNotClearPreviousOrders(t *testing.T) {
	g := orderGame(t, `{"wood":0}`, "")
	p := g.Human(0)
	id := unitIDs(g, p, 1)[0]
	g.Submit(p, harness.OrderMove([]uint64{id}, 3, 0))
	g.EndTurn()
	old := g.Self(p).ActiveOrders[0].InternalID
	build := harness.OrderBuild([]uint64{id}, 2, 1, 0, 0, 0)
	build.Clear_previous_orders = true
	g.Submit(p, build)
	g.EndTurn()
	if orders := g.State(p).Unit(id).ActiveOrders; len(orders) != 1 || orders[0].InternalID != old {
		t.Fatalf("rejected replacement cleared queue: %v", orders)
	}
	if refusals := g.Self(p).Refusals(); len(refusals) != 1 || refusals[0] != "cannot afford building" {
		t.Fatalf("refusals: %v", refusals)
	}
}
