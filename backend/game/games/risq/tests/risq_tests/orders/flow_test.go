package orders

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

// Cancelling, clearing, replacing, deleting and unsubmitting orders on independent units over five turns
func TestOrderFlow(t *testing.T) {
	g := newGame(t, layout{workers: 8})
	p, e := g.Human(0), g.Human(1)
	mine := scan(g, p)
	v, b := mine.workers, mine.buildings
	carried, shared, queued, cleared, deleted, spare := v[0], v[1:3], v[3], v[4:6], v[6], v[7]
	far := func(ids ...uint64) defs.OrderFromFrontend { return harness.OrderMove(ids, 3, 0) }

	g.Submit(p,
		far(carried), far(shared...), far(queued), harness.OrderMove([]uint64{queued}, 0, 0),
		far(cleared...), harness.Order(defs.OrderType_UnitDelete, cleared[:1], 0, false),
		far(deleted), harness.Order(defs.OrderType_UnitDelete, []uint64{deleted}, 0, true),
		far(spare),
		harness.Order(defs.OrderType_BuildingResearch, []uint64{b[201]}, 301, false), harness.OrderProduce([]uint64{b[201]}, 11),
		harness.Order(defs.OrderType_BuildingDelete, []uint64{b[201]}, 0, false),
	)
	if len(unitByID(g, p, carried).ActiveOrders) != 0 {
		t.Fatal("order delivered before all players submitted")
	}
	g.EndTurn()

	// turn 1: orders are received, carried and stable; delete with clear removes the unit and its move
	if hasUnit(g, p, deleted) {
		t.Fatal("deleted unit or its cleared orders remained")
	}
	carriedOrder := unitByID(g, p, carried).ActiveOrders
	sharedOrder := unitByID(g, p, shared[0]).ActiveOrders
	queue := unitByID(g, p, queued).ActiveOrders
	if len(carriedOrder) != 1 || carriedOrder[0].InternalID == 0 || len(sharedOrder) != 1 || len(queue) != 2 || len(activeOrderIDs(g, p, uint8(defs.OrderType_BuildingDelete))) != 1 {
		t.Fatalf("turn one orders: carried %+v shared %+v queue %+v", carriedOrder, sharedOrder, queue)
	}
	if unitByID(g, p, carried).Space == (harness.Coord{X: 3}) {
		t.Fatal("move unexpectedly completed in one turn")
	}
	stopped := unitByID(g, p, shared[0])

	// unsubmit discards only the new orders and never charges for pending production
	before := g.Self(p)
	soldiers := len(unitIDs(g, p, 21))
	g.Submit(p, harness.OrderMoveZone([]uint64{spare}, 1, 0, 0, 0), harness.OrderProduce([]uint64{b[205]}, 21))
	pending := g.Self(p)
	if len(pending.ActiveOrders) != len(before.ActiveOrders) || len(pending.UnreceivedOrders) != 2 || pending.Resources.Food != before.Resources.Food {
		t.Fatalf("pending orders were not stored unreceived without charging: %+v", pending)
	}
	g.Action(p, "unsubmit-orders", nil)
	if after := g.Self(p); after.OrdersSubmitted || len(after.ActiveOrders) != len(before.ActiveOrders) || len(after.UnreceivedOrders) != 0 || len(unitByID(g, p, spare).ActiveOrders) != 1 {
		t.Fatalf("unsubmit lost received orders or kept new ones: %+v", after)
	}

	// another player cannot cancel this player's order
	g.SubmitRejected(e, harness.Order(defs.OrderType_CancelOrder, nil, int64(carriedOrder[0].InternalID), false), fmt.Sprintf("No active order with id %d", carriedOrder[0].InternalID))

	clearProduce := harness.OrderProduce([]uint64{b[201]}, 11)
	clearProduce.Clear_previous_orders = true
	g.Submit(p,
		harness.Order(defs.OrderType_CancelOrder, shared[:1], int64(sharedOrder[0].InternalID), false),
		harness.Order(defs.OrderType_CancelOrder, nil, int64(queue[1].InternalID), false),
		harness.Order(defs.OrderType_UnitMoveSpace, cleared[:1], harness.SpaceKey(0, 0), true),
		clearProduce,
	)
	g.EndTurn()

	// turn 2: each cancel or clear touched only its own subject
	if u := unitByID(g, p, shared[0]); u.Space != stopped.Space || u.Zone != stopped.Zone || len(u.ActiveOrders) != 0 {
		t.Fatalf("cancelled subject kept moving: %+v", u)
	}
	if u := unitByID(g, p, shared[1]); len(u.ActiveOrders) != 1 || u.ActiveOrders[0].InternalID != sharedOrder[0].InternalID {
		t.Fatalf("other subject lost shared order: %+v", u)
	}
	if got := unitByID(g, p, queued).ActiveOrders; len(got) != 1 || got[0].InternalID != queue[0].InternalID {
		t.Fatalf("wrong queued order cancelled: %v", got)
	}
	if u := unitByID(g, p, cleared[0]); u.Space != (harness.Coord{}) || len(u.ActiveOrders) != 0 {
		t.Fatalf("replacement did not clear the queue: %+v", u)
	}
	if u := unitByID(g, p, cleared[1]); len(u.ActiveOrders) != 1 {
		t.Fatalf("unselected subject lost its order: %+v", u)
	}
	if res := g.Self(p).Resources; !g.Self(p).ResearchedTechs[301] || len(activeOrderIDs(g, p, uint8(defs.OrderType_BuildingDelete))) != 0 || res.Food != 860 || res.Wood != 960 {
		t.Fatalf("clear cancelled or refunded immune work, or preserved cancellable building work: %+v", res)
	}
	g.Submit(p, harness.Order(defs.OrderType_CancelOrder, nil, int64(sharedOrder[0].InternalID), false))
	g.EndTurn()

	// turn 3: whole-order cancellation leaves no active work
	for _, id := range activeOrderIDs(g, p, uint8(defs.OrderType_UnitMoveSpace)) {
		if id == sharedOrder[0].InternalID {
			t.Fatal("whole-order cancellation left active work")
		}
	}
	halted := unitByID(g, p, shared[1])
	g.EndTurn()
	g.EndTurn()

	// turns 4 and 5: cancelled moves never restart, uncancelled ones finish
	for _, was := range []harness.Unit{stopped, halted} {
		if u := unitByID(g, p, was.InternalID); u.Space != was.Space || u.Zone != was.Zone {
			t.Fatalf("cancelled move restarted: %+v", u)
		}
	}
	if u := unitByID(g, p, queued); u.Space != (harness.Coord{X: 3}) || len(u.ActiveOrders) != 0 {
		t.Fatalf("cancelled return move executed or front move unfinished: %+v", u)
	}
	if u := unitByID(g, p, carried); u.Space != (harness.Coord{X: 3}) || len(u.ActiveOrders) != 0 || g.Self(p).OrdersSubmitted {
		t.Fatalf("carried move did not finish and reset: %+v", u)
	}
	g.SubmitRejected(p, harness.Order(defs.OrderType_CancelOrder, nil, int64(queue[1].InternalID), false), fmt.Sprintf("No active order with id %d", queue[1].InternalID))
	if len(unitIDs(g, p, 21)) != soldiers {
		t.Fatal("discarded production executed")
	}
	if got := len(unitIDs(g, p, 11)); got != 9 {
		t.Fatalf("%d workers, want 9: preserved production did not complete", got)
	}
}
