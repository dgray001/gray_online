package combat

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
	"github.com/gin-gonic/gin"
	"testing"
)

func TestDefensiveClearsPriorityCategoriesInOrder(t *testing.T) {
	doc := `{"board_size":1,"players":2,"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":1,"y":0,"building":{"id":2,"player":1},"units":[{"id":13,"player":0,"count":1},{"id":11,"player":1,"count":1},{"id":1,"player":1,"count":1}]},{"x":-1,"y":0,"units":[{"id":11,"player":1,"count":1}]}]}]}`
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"priority-sequence": doc})
	g := harness.NewGame(t, "custom:priority-sequence", 1, 2)
	owner, enemy := g.Human(0), g.Human(1)
	g.Action(owner, "set-unit-behavior", gin.H{"internal_ids": []uint64{g.Self(owner).Units[0].InternalID}, "stance": uint8(defs.UnitStance_DEFENSIVE), "attack_back": false, "target_priority": []uint8{uint8(defs.TargetCategory_MILITARY), uint8(defs.TargetCategory_ECONOMIC), uint8(defs.TargetCategory_BUILDING)}})
	for _, unit := range g.Self(enemy).Units {
		if unit.UnitID == 11 {
			g.Action(enemy, "set-unit-behavior", gin.H{"internal_ids": []uint64{unit.InternalID}, "stance": uint8(defs.UnitStance_PASSIVE), "attack_back": false})
		}
	}
	for turn := 0; turn < 8; turn++ {
		g.EndTurn()
		state := g.Self(enemy)
		military, villagers := 0, 0
		for _, unit := range state.Units {
			if unit.UnitID == 11 {
				military++
			} else {
				villagers++
			}
		}
		if military > 0 {
			if villagers != 1 {
				t.Fatal("villager killed before all military targets")
			}
			for _, unit := range state.Units {
				if unit.UnitID == 1 && unit.CombatStats.Health != float64(unit.CombatStats.MaxHealth) {
					t.Fatal("villager damaged before all military targets")
				}
			}
		}
		building := state.Buildings[0]
		if military+villagers > 0 && building.CombatStats.Health != float64(building.CombatStats.MaxHealth) {
			t.Fatal("building damaged before all unit targets")
		}
		if military+villagers == 0 && building.CombatStats.Health < float64(building.CombatStats.MaxHealth) {
			return
		}
	}
	t.Fatal("did not progress from military to villagers to buildings")
}
