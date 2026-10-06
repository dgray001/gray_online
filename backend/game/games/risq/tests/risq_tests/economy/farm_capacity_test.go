package economy

import (
	"testing"

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
