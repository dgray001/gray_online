package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func TestContestedGatherSatisfiesSmallerDemandFirst(t *testing.T) {
	for _, slots := range [][2]int{{0, 1}, {1, 0}} {
		doc := `{"board_size":1,"players":2,"starting_bank":{},"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"resource":11,"units":[{"id":1,"player":0,"count":1},{"id":2,"player":1,"count":1}]}]}]}`
		fakeboard.UseConfig(t, shippedConfig, nil, map[string]string{"unequal": doc})
		worker := defs.UnitConfigs[1]
		worker.Turn_stamina = 1
		defs.UnitConfigs[2] = worker
		resource := defs.ResourceConfigs[11]
		resource.Starting_resources, resource.Base_gather_speed = 4, 10
		defs.ResourceConfigs[11] = resource
		g := harness.NewGame(t, "custom:unequal", 1, 2)
		for _, slot := range slots {
			submitGather(g, g.Human(slot))
		}
		for slot, want := range []float64{3, 1} {
			state := g.Self(g.Human(slot))
			if state.Resources.Wood != want || len(state.Refusals()) != 0 {
				t.Errorf("submission order %v, slot %d: wood %v, refusals %v; want %v wood", slots, slot, state.Resources.Wood, state.Refusals(), want)
			}
		}
		space := g.State(g.Human(0)).Space(0, 0)
		if len(space.Resources) != 0 {
			t.Error("fully gathered resource remains on the board")
		}
	}
}
