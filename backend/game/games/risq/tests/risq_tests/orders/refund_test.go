package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

// A refund frees what its replacement spends, wherever each sits in the batch, and duplicate refunds pay once
func TestRefundsPrecedeSpending(t *testing.T) {
	for _, refundFirst := range []bool{false, true} {
		g := newGame(t, layout{bank: `{"food":70,"wood":70,"stone":100,"gold":100}`, villagers: 3})
		p := g.Human(0)
		mine := scan(g, p)
		v, b := mine.villagers, mine.buildings
		g.Submit(p,
			harness.OrderProduce([]uint64{b[22]}, 12),
			harness.Order(defs.OrderType_BuildingResearch, []uint64{b[11]}, 2, false),
			harness.OrderBuild(v[:1], 2, 3, 0, 0, 0),
			harness.OrderMove(v[1:2], 3, 0), harness.OrderMove(v[1:2], 0, 0),
		)
		g.EndTurn()
		state := g.Self(p)
		production := activeOrderIDs(g, p, uint8(defs.OrderType_BuildingCreate))
		researching := activeOrderIDs(g, p, uint8(defs.OrderType_BuildingResearch))
		if len(state.ActiveOrders) != 5 || len(state.PlannedFoundations) != 1 || *state.Resources != (harness.Resources{}) {
			t.Fatalf("refund setup is not fully reserved: %+v", state)
		}
		cancelProduction := harness.Order(defs.OrderType_CancelOrder, nil, int64(production[0]), false)
		spend := []defs.OrderFromFrontend{
			harness.OrderProduce([]uint64{b[22]}, 12),
			harness.Order(defs.OrderType_BuildingResearch, []uint64{b[11]}, 3, false),
			harness.OrderBuild(v[:1], 2, 0, 0, -1, 1),
		}
		refund := []defs.OrderFromFrontend{
			cancelProduction, cancelProduction,
			harness.Order(defs.OrderType_CancelOrder, nil, int64(researching[0]), false),
			harness.Order(defs.OrderType_CancelFoundation, nil, harness.ZoneKey(3, 0, 0, 0), false),
		}
		batch := append(append([]defs.OrderFromFrontend{}, spend...), refund...)
		if refundFirst {
			batch = append(append([]defs.OrderFromFrontend{}, refund...), spend...)
		}
		g.Submit(p, batch...)
		g.EndTurn()
		state = g.Self(p)
		if len(state.Refusals()) != 0 || len(state.ActiveOrders) != 5 || *state.Resources != (harness.Resources{}) {
			t.Fatalf("refund-first=%v: refunds did not fund the replacements exactly once: %+v", refundFirst, state)
		}
		for _, order := range state.ActiveOrders {
			if order.InternalID == production[0] || order.InternalID == researching[0] {
				t.Fatalf("refund-first=%v: a refunded order survived", refundFirst)
			}
		}
		if len(activeOrderIDs(g, p, uint8(defs.OrderType_BuildingCreate))) != 1 || len(activeOrderIDs(g, p, uint8(defs.OrderType_BuildingResearch))) != 1 || len(activeOrderIDs(g, p, uint8(defs.OrderType_UnitBuild))) != 1 {
			t.Fatalf("refund-first=%v: replacements are not exactly one of each kind", refundFirst)
		}
		if refundFirst {
			continue
		}

		// an unaffordable replacement leaves the unit's previous orders alone
		oldMove := unitByID(g, p, v[1]).ActiveOrders
		rebuild := harness.OrderBuild(v[1:2], 2, 1, 0, 0, 0)
		rebuild.Clear_previous_orders = true
		g.Submit(p, rebuild)
		g.EndTurn()
		got := unitByID(g, p, v[1]).ActiveOrders
		if len(got) == 0 || got[len(got)-1].InternalID != oldMove[len(oldMove)-1].InternalID {
			t.Fatalf("rejected replacement cleared the queue: %v", got)
		}
		if refusals := g.Self(p).Refusals(); len(refusals) != 1 || refusals[0] != "cannot afford building" {
			t.Fatalf("refusals: %v", refusals)
		}
	}
}

// Cancelling one subject of a shared production refunds only that subject's reservation
func TestCancelOneProducerRefundsOnlyItsReservation(t *testing.T) {
	g := newGame(t, layout{bank: `{"food":140,"wood":80}`, tweak: func() {
		center := defs.BuildingConfigs[1]
		center.Produces = append(append([]defs.Producible(nil), center.Produces...), defs.Producible{Kind: defs.ProducibleKind_UNIT, Id: 12})
		defs.BuildingConfigs[1] = center
	}})
	p := g.Human(0)
	b := scan(g, p).buildings
	g.Submit(p, harness.OrderProduce([]uint64{b[1], b[22]}, 12))
	g.EndTurn()
	state := g.Self(p)
	if len(state.ActiveOrders) != 1 || state.Resources.Food != 0 || state.Resources.Wood != 0 {
		t.Fatal("setup did not reserve both productions")
	}
	g.Submit(p, harness.Order(defs.OrderType_CancelOrder, []uint64{b[22]}, int64(state.ActiveOrders[0].InternalID), false))
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
