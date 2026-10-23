package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestOrchardCapacityLimitsHarvest(t *testing.T) {
	g := orchardGame(t, 120, 3)
	p := g.Human(0)
	units := g.Self(p).Units
	ids := make([]uint64, len(units))
	for i, u := range units {
		ids[i] = u.InternalID
	}
	g.Submit(p, harness.OrderGather(ids, 0, 0, 0, 0))
	g.EndTurn()
	want := float64(units[0].CurrentStamina) * 2 * 0.6
	if state := g.Self(p); state.Resources.Food != want || orchardZone(g, p).Building.ResourcesLeft != 150-want {
		t.Errorf("orchard harvest %+v, pool %v; want %v food", state.Resources, orchardZone(g, p).Building.ResourcesLeft, want)
	}
	idle := 0
	for _, u := range g.Self(p).Units {
		if u.CurrentStamina > units[0].CurrentStamina {
			idle++
		}
	}
	if idle != 1 {
		t.Errorf("%d workers retained unused stamina, want one", idle)
	}
}

func TestOrchardCompletionAutoGatherEnroute(t *testing.T) {
	spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":0,"count":1},{"id":21,"player":0,"count":1}]}]},{"x":1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":0,"count":1}]}]}`
	g := economyGame(t, `{"wood":120}`, spaces+","+remoteWorker, `,"starting_techs":[301]`)
	p := g.Human(0)
	g.Submit(p, harness.OrderBuild(workerIds(g, p), 203, 0, 0, 0, 0))
	g.EndTurn()
	g.EndTurn()
	orders := g.Self(p).ActiveOrders
	if len(orders) != 2 {
		t.Fatalf("expected 2 active gather orders, got %d", len(orders))
	}
	for _, o := range orders {
		if o.OrderType != uint8(defs.OrderType_UnitGather) {
			t.Errorf("expected OrderType_UnitGather, got %d", o.OrderType)
		}
	}
}

func TestOrchardCompletionMultiOrderSkipped(t *testing.T) {
	space := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":0,"count":1},{"id":21,"player":0,"count":1}]}]}`
	g := economyGame(t, `{"wood":120}`, space+","+remoteWorker, `,"starting_techs":[301]`)
	p := g.Human(0)
	v := workerIds(g, p)[0]
	g.Submit(p, harness.OrderBuild([]uint64{v}, 203, 0, 0, 0, 0), harness.OrderMoveZone([]uint64{v}, 0, 0, 1, 0))
	g.EndTurn()
	g.EndTurn()
	for _, o := range g.Self(p).ActiveOrders {
		if o.OrderType == uint8(defs.OrderType_UnitGather) {
			t.Errorf("worker with queued follow-up order was auto-assigned gather: %+v", o)
		}
	}
}
