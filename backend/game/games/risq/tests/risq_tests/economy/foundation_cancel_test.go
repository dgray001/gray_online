package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestOnlyFoundationCancellationRefundsReservation(t *testing.T) {
	g := buildGame(t, 1)
	p := g.Human(0)
	wood := g.Self(p).Resources.Wood
	g.Submit(p, housingOrders(g, p, 2)...)
	g.EndTurn()
	state := g.Self(p)
	if len(state.Buildings) != 0 || len(state.PlannedFoundations) != 1 || len(state.ActiveOrders) != 1 || state.Resources.Wood != wood-30 {
		t.Fatalf("setup: buildings %v, foundations %v, orders %v, wood %v; want one reserved build while traveling", state.Buildings, state.PlannedFoundations, state.ActiveOrders, state.Resources.Wood)
	}
	order := state.ActiveOrders[0]
	foundation := state.PlannedFoundations[0]
	if order.OrderType != uint8(defs.OrderType_UnitBuild) || foundation.BuildingID != 2 || foundation.CoordinateKey != uint(harness.ZoneKey(2, 0, 0, 0)) {
		t.Fatal("reservation or active order does not match the requested housing")
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
