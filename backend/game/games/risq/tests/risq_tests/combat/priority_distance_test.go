package combat

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
	"github.com/gin-gonic/gin"
)

func priorityDistanceGame(t *testing.T) *harness.Game {
	t.Helper()
	doc := `{"board_size":1,"players":2,"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":1,"y":0,"building":{"id":2,"player":1},"units":[{"id":11,"player":0,"count":1},{"id":1,"player":1,"count":1}]},{"x":-1,"y":0,"units":[{"id":13,"player":1,"count":1}]}]}]}`
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"priority-distance": doc})
	return harness.NewGame(t, "custom:priority-distance", 1, 2)
}

func TestTargetPriorityPrecedesDistance(t *testing.T) {
	for _, c := range []struct {
		name     string
		stance   defs.UnitStance
		explicit bool
	}{
		{"defensive", defs.UnitStance_DEFENSIVE, false},
		{"aggressive", defs.UnitStance_AGGRESSIVE, false},
		{"attack-space", defs.UnitStance_PASSIVE, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			g := priorityDistanceGame(t)
			owner, enemy := g.Human(0), g.Human(1)
			attacker := g.Self(owner).Units[0]
			g.Action(owner, "set-unit-behavior", gin.H{"internal_ids": []uint64{attacker.InternalID}, "stance": uint8(c.stance), "attack_back": false, "target_priority": []uint8{uint8(defs.TargetCategory_MILITARY), uint8(defs.TargetCategory_ECONOMIC), uint8(defs.TargetCategory_BUILDING)}})
			for _, unit := range g.Self(enemy).Units {
				if unit.UnitID == 13 {
					g.Action(enemy, "set-unit-behavior", gin.H{"internal_ids": []uint64{unit.InternalID}, "stance": uint8(defs.UnitStance_PASSIVE), "attack_back": false})
				}
			}
			if c.explicit {
				g.Submit(owner, harness.Order(defs.OrderType_UnitAttackSpace, []uint64{attacker.InternalID}, harness.SpaceKey(0, 0), false))
			} else {
				g.Submit(owner)
			}
			g.Submit(enemy)
			state := g.Self(enemy)
			if len(state.Units) != 2 {
				t.Fatalf("lower-priority unit killed first: %+v", state.Units)
			}
			for _, unit := range state.Units {
				if unit.UnitID == 13 && unit.CombatStats.Health >= float64(unit.CombatStats.MaxHealth) {
					t.Fatal("distant military target was ignored")
				}
				if unit.UnitID == 1 && unit.CombatStats.Health != float64(unit.CombatStats.MaxHealth) {
					t.Fatal("nearby villager was attacked first")
				}
			}
			if building := state.Buildings[0]; building.CombatStats.Health != float64(building.CombatStats.MaxHealth) {
				t.Fatal("nearby building was attacked first")
			}
		})
	}
}
