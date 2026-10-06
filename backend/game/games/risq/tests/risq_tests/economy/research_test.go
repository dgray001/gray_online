package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestResearchAppliesToExistingAndNewUnits(t *testing.T) {
	g := productionGame(t, 1, 1, "")
	p := g.Human(0)
	b := centers(g, p)[0]
	g.Submit(p, harness.Order(defs.OrderType_BuildingResearch, []uint64{b.InternalID}, 1, false))
	g.EndTurn()
	state := g.Self(p)
	if state.ResearchedTechs[1] || state.Units[0].CombatStats.MaxHealth != 8 || state.Resources.Food != 250 || state.Resources.Wood != 250 {
		t.Fatalf("unfinished research applied stats early or charged incorrectly: %+v", state)
	}
	g.EndTurn()
	if !g.Self(p).ResearchedTechs[1] {
		t.Fatal("farming did not complete on turn two")
	}
	g.Submit(p, harness.OrderProduce([]uint64{b.InternalID}, 1))
	g.EndTurn()
	state = g.Self(p)
	if len(state.Units) != 2 || state.Resources.Food != 200 || state.Resources.Wood != 250 {
		t.Fatalf("research and production state: units %d, resources %+v", len(state.Units), state.Resources)
	}
	for _, u := range state.Units {
		if u.TurnStamina != 10 || u.CombatStats.MaxHealth != 14 || u.CombatStats.Health != 14 || u.CombatStats.DefenseBlunt != 1 {
			t.Errorf("farming bonuses missing or duplicated: %+v", u)
		}
	}
}
