package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestSameTickResearchAndProductionApplyBonusesOnce(t *testing.T) {
	for _, researcher := range []int{0, 1} {
		g := productionGame(t, 1, 2, "")
		p := g.Human(0)
		original := defs.TechConfigs[1]
		config := original
		config.Research_stamina = 10
		defs.TechConfigs[1] = config
		t.Cleanup(func() { defs.TechConfigs[1] = original })
		buildings := centers(g, p)
		g.Submit(p, harness.Order(defs.OrderType_BuildingResearch, []uint64{buildings[researcher].InternalID}, 1, false), harness.OrderProduce([]uint64{buildings[1-researcher].InternalID}, 1))
		g.EndTurn()
		state := g.Self(p)
		if !state.ResearchedTechs[1] || len(state.Units) != 2 || state.Resources.Food != 200 || state.Resources.Wood != 250 {
			t.Fatalf("same-tick research and production did not finish: %+v", state)
		}
		for _, u := range state.Units {
			if u.CombatStats.MaxHealth != 14 || u.CombatStats.Health != 14 || u.TurnStamina != 10 || u.CombatStats.DefenseBlunt != 1 {
				t.Errorf("same-tick bonus missing or duplicated: %+v", u)
			}
		}
	}
}
