package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestFarmCapacityLimitsHarvest(t *testing.T) {
	g := farmGame(t, 120, 3)
	p := g.Human(0)
	units := g.Self(p).Units
	ids := make([]uint64, len(units))
	for i, u := range units {
		ids[i] = u.InternalID
	}
	g.Submit(p, harness.OrderGather(ids, 0, 0, 0, 0))
	g.EndTurn()
	want := float64(units[0].CurrentStamina) * 2 * 0.8
	if state := g.Self(p); state.Resources.Food != want || farmZone(g, p).Building.ResourcesLeft != 200-want {
		t.Errorf("farm harvest %+v, pool %v; want %v food", state.Resources, farmZone(g, p).Building.ResourcesLeft, want)
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

func TestFarmCompletionAutoGatherEnroute(t *testing.T) {
	spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":0,"count":1},{"id":101,"player":0,"count":1}]}]},{"x":1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":0,"count":1}]}]}`
	g := economyGame(t, `{"wood":120}`, spaces+","+remoteVillager, `,"starting_techs":[1]`)
	p := g.Human(0)
	g.Submit(p, harness.OrderBuild(villagerIds(g, p), 3, 0, 0, 0, 0))
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

func TestFarmCompletionMultiOrderSkipped(t *testing.T) {
	space := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":0,"count":1},{"id":101,"player":0,"count":1}]}]}`
	g := economyGame(t, `{"wood":120}`, space+","+remoteVillager, `,"starting_techs":[1]`)
	p := g.Human(0)
	v := villagerIds(g, p)[0]
	g.Submit(p, harness.OrderBuild([]uint64{v}, 3, 0, 0, 0, 0), harness.OrderMoveZone([]uint64{v}, 0, 0, 1, 0))
	g.EndTurn()
	g.EndTurn()
	for _, o := range g.Self(p).ActiveOrders {
		if o.OrderType == uint8(defs.OrderType_UnitGather) {
			t.Errorf("worker with queued follow-up order was auto-assigned gather: %+v", o)
		}
	}
}
