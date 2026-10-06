package economy

import (
	"math"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestGatherCapacityLimitsHarvest(t *testing.T) {
	g := sourceGame(t, false)
	p := g.Human(0)
	units := g.Self(p).Units
	subjects := make([]uint64, len(units))
	for i, u := range units {
		subjects[i] = u.InternalID
	}
	config := defs.ResourceConfigs[11]
	before := g.Self(p).Resources.Wood
	want := float64(units[0].CurrentStamina*config.Gather_capacity) * float64(config.Base_gather_speed) / 10
	g.Submit(p, harness.OrderGather(subjects, 0, 0, 0, 0))
	g.EndTurn()
	if got := g.Self(p).Resources.Wood - before; math.Abs(got-want) > 0.00001 {
		t.Errorf("three workers gained %v wood, want capacity-limited %v", got, want)
	}
	left := g.State(p).Space(0, 0).Resources[0].ResourcesLeft
	if math.Abs(left-(config.Starting_resources-want)) > 0.00001 {
		t.Errorf("cedar pool %v, want %v", left, config.Starting_resources-want)
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
