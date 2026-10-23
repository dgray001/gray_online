package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestFoundationRaceRefundsLoser(t *testing.T) {
	winners := []int{}
	for _, slots := range [][2]int{{0, 1}, {1, 0}} {
		g := scenarioGame(t, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":0,"count":1},{"id":11,"player":1,"count":1}]}]},{"x":-1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":202,"player":0}}]},{"x":1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":202,"player":1}}]}`)
		for _, slot := range slots {
			p := g.Human(slot)
			g.Submit(p, shelterOrders(g, p, 0)...)
		}
		winner, created := -1, 0
		for slot := 0; slot < 2; slot++ {
			state := g.Self(g.Human(slot))
			owns := false
			for _, b := range state.Buildings {
				if b.Space == (harness.Coord{}) {
					winner, owns = slot, true
					created++
				}
			}
			wantWood := 120.0
			if owns {
				wantWood -= 30
			}
			if state.Resources.Wood != wantWood || len(state.PlannedFoundations) != 0 {
				t.Errorf("slot %d: wood %v, foundations %v; want %v wood and no reservation", slot, state.Resources.Wood, state.PlannedFoundations, wantWood)
			}
			if !owns && len(state.ActiveOrders) != 0 {
				t.Errorf("loser retained orders %v", state.ActiveOrders)
			}
		}
		if created != 1 {
			t.Fatalf("created %d buildings at the contested zone, want one", created)
		}
		winners = append(winners, winner)
		g.EndTurn()
		if state := g.Self(g.Human(1 - winner)); state.Resources.Wood != 120 || len(state.Buildings) != 1 {
			t.Error("loser was charged again or created another building")
		}
	}
	if winners[0] != winners[1] {
		t.Errorf("submission order changed winner: %v", winners)
	}
}

func TestOnlyFoundationCancellationRefundsReservation(t *testing.T) {
	g := buildGame(t, 1)
	p := g.Human(0)
	wood := g.Self(p).Resources.Wood
	g.Submit(p, shelterOrders(g, p, 2)...)
	g.EndTurn()
	state := g.Self(p)
	if len(state.Buildings) != 0 || len(state.PlannedFoundations) != 1 || len(state.ActiveOrders) != 1 || state.Resources.Wood != wood-30 {
		t.Fatalf("setup: buildings %v, foundations %v, orders %v, wood %v; want one reserved build while traveling", state.Buildings, state.PlannedFoundations, state.ActiveOrders, state.Resources.Wood)
	}
	order := state.ActiveOrders[0]
	foundation := state.PlannedFoundations[0]
	if order.OrderType != uint8(defs.OrderType_UnitBuild) || foundation.BuildingID != 202 || foundation.CoordinateKey != uint(harness.ZoneKey(2, 0, 0, 0)) {
		t.Fatal("reservation or active order does not match the requested shelter")
	}
	g.Submit(p, harness.Order(defs.OrderType_CancelOrder, nil, int64(order.InternalID), false))
	g.EndTurn()
	state = g.Self(p)
	if len(state.ActiveOrders) != 0 || len(state.Buildings) != 0 || len(state.PlannedFoundations) != 1 || state.Resources.Wood != wood-30 {
		t.Fatalf("order cancellation: buildings %v, foundations %v, orders %v, wood %v; want reservation and cost retained", state.Buildings, state.PlannedFoundations, state.ActiveOrders, state.Resources.Wood)
	}
	g.EndTurn()
	if state = g.Self(p); len(state.Buildings) != 0 || len(state.PlannedFoundations) != 1 || state.Resources.Wood != wood-30 {
		t.Fatal("cancelled builder changed the reservation on an idle turn")
	}
	g.Submit(p, harness.Order(defs.OrderType_CancelFoundation, nil, harness.ZoneKey(2, 0, 0, 0), false))
	g.EndTurn()
	state = g.Self(p)
	if len(state.PlannedFoundations) != 0 || len(state.Buildings) != 0 || state.Resources.Wood != wood {
		t.Errorf("foundation cancellation: foundations %v, buildings %v, wood %v; want no foundation and %v wood", state.PlannedFoundations, state.Buildings, state.Resources.Wood, wood)
	}
	g.EndTurn()
	if got := g.Self(p).Resources.Wood; got != wood {
		t.Errorf("refund repeated: wood %v, want %v", got, wood)
	}
}
