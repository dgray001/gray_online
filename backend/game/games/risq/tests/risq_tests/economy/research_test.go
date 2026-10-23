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
	g.Submit(p, harness.Order(defs.OrderType_BuildingResearch, []uint64{b.InternalID}, 301, false))
	g.EndTurn()
	state := g.Self(p)
	if state.ResearchedTechs[301] || state.Units[0].CombatStats.MaxHealth != 10 || state.Resources.Food != 260 || state.Resources.Wood != 260 {
		t.Fatalf("unfinished research applied stats early or charged incorrectly: %+v", state)
	}
	g.EndTurn()
	if !g.Self(p).ResearchedTechs[301] {
		t.Fatal("labor tech did not complete on turn two")
	}
	g.Submit(p, harness.OrderProduce([]uint64{b.InternalID}, 11))
	g.EndTurn()
	state = g.Self(p)
	if len(state.Units) != 2 || state.Resources.Food != 210 || state.Resources.Wood != 260 {
		t.Fatalf("research and production state: units %d, resources %+v", len(state.Units), state.Resources)
	}
	for _, u := range state.Units {
		if u.TurnStamina != 14 || u.CombatStats.MaxHealth != 20 || u.CombatStats.Health != 20 || u.CombatStats.DefenseBlunt != 2 {
			t.Errorf("labor tech bonuses missing or duplicated: %+v", u)
		}
	}
}

func TestResearchCancellationRefundsAndAllowsRetry(t *testing.T) {
	g := productionGame(t, 1, 1, "")
	p := g.Human(0)
	b := centers(g, p)[0]
	research := harness.Order(defs.OrderType_BuildingResearch, []uint64{b.InternalID}, 301, false)
	g.Submit(p, research)
	g.EndTurn()
	orders := g.Self(p).ActiveOrders
	if len(orders) != 1 {
		t.Fatalf("research orders %v, want one", orders)
	}
	g.Submit(p, harness.Order(defs.OrderType_CancelOrder, nil, int64(orders[0].InternalID), false))
	g.EndTurn()
	state := g.Self(p)
	if state.Resources.Food != 300 || state.Resources.Wood != 300 || state.ResearchedTechs[301] || state.Units[0].CombatStats.MaxHealth != 10 || len(centers(g, p)[0].ProductionQueue) != 0 {
		t.Fatalf("cancelled research retained cost, bonuses, or queue: %+v", state)
	}
	g.Submit(p, research)
	g.EndTurn()
	g.EndTurn()
	state = g.Self(p)
	if !state.ResearchedTechs[301] || state.Resources.Food != 260 || state.Resources.Wood != 260 || state.Units[0].CombatStats.MaxHealth != 20 {
		t.Errorf("retried research did not finish with one charge: %+v", state)
	}
}

func TestSameTickResearchAndProductionApplyBonusesOnce(t *testing.T) {
	for _, researcher := range []int{0, 1} {
		g := productionGame(t, 1, 2, "")
		p := g.Human(0)
		original := defs.TechConfigs[301]
		config := original
		config.Research_stamina = 10
		defs.TechConfigs[301] = config
		t.Cleanup(func() { defs.TechConfigs[301] = original })
		buildings := centers(g, p)
		g.Submit(p, harness.Order(defs.OrderType_BuildingResearch, []uint64{buildings[researcher].InternalID}, 301, false), harness.OrderProduce([]uint64{buildings[1-researcher].InternalID}, 11))
		g.EndTurn()
		state := g.Self(p)
		if !state.ResearchedTechs[301] || len(state.Units) != 2 || state.Resources.Food != 210 || state.Resources.Wood != 260 {
			t.Fatalf("same-tick research and production did not finish: %+v", state)
		}
		for _, u := range state.Units {
			if u.CombatStats.MaxHealth != 20 || u.CombatStats.Health != 20 || u.TurnStamina != 14 || u.CombatStats.DefenseBlunt != 2 {
				t.Errorf("same-tick bonus missing or duplicated: %+v", u)
			}
		}
	}
}

func TestResearchLeavesNonmatchingAndEnemyUnitsUnchanged(t *testing.T) {
	g := economyGame(t, `{"food":100,"wood":100}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":201,"player":0},"units":[{"id":11,"player":0,"count":1},{"id":21,"player":0,"count":1}]}]},`+remoteWorker, "")
	p, enemy := g.Human(0), g.Human(1)
	before := g.Self(p).Units
	enemyBefore := g.Self(enemy).Units[0]
	g.Submit(p, harness.Order(defs.OrderType_BuildingResearch, []uint64{g.Self(p).Buildings[0].InternalID}, 301, false))
	g.EndTurn()
	g.EndTurn()
	if !g.Self(p).ResearchedTechs[301] {
		t.Fatal("labor tech did not finish")
	}
	for _, u := range before {
		if u.UnitID == 21 {
			after := g.State(p).Unit(u.InternalID)
			if after == nil || after.CombatStats != u.CombatStats || after.TurnStamina != u.TurnStamina {
				t.Error("labor tech affected nonmatching grunt")
			}
		}
	}
	if after := g.Self(enemy).Units[0]; after.CombatStats != enemyBefore.CombatStats || after.TurnStamina != enemyBefore.TurnStamina || g.Self(enemy).ResearchedTechs[301] {
		t.Error("labor tech affected the enemy")
	}
}
