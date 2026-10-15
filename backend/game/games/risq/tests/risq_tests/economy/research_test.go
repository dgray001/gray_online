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

func TestResearchCancellationRefundsAndAllowsRetry(t *testing.T) {
	g := productionGame(t, 1, 1, "")
	p := g.Human(0)
	b := centers(g, p)[0]
	research := harness.Order(defs.OrderType_BuildingResearch, []uint64{b.InternalID}, 1, false)
	g.Submit(p, research)
	g.EndTurn()
	orders := g.Self(p).ActiveOrders
	if len(orders) != 1 {
		t.Fatalf("research orders %v, want one", orders)
	}
	g.Submit(p, harness.Order(defs.OrderType_CancelOrder, nil, int64(orders[0].InternalID), false))
	g.EndTurn()
	state := g.Self(p)
	if state.Resources.Food != 300 || state.Resources.Wood != 300 || state.ResearchedTechs[1] || state.Units[0].CombatStats.MaxHealth != 8 || len(centers(g, p)[0].ProductionQueue) != 0 {
		t.Fatalf("cancelled research retained cost, bonuses, or queue: %+v", state)
	}
	g.Submit(p, research)
	g.EndTurn()
	g.EndTurn()
	state = g.Self(p)
	if !state.ResearchedTechs[1] || state.Resources.Food != 250 || state.Resources.Wood != 250 || state.Units[0].CombatStats.MaxHealth != 14 {
		t.Errorf("retried research did not finish with one charge: %+v", state)
	}
}

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

func TestResearchLeavesNonmatchingAndEnemyUnitsUnchanged(t *testing.T) {
	g := economyGame(t, `{"food":100,"wood":100}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":0},"units":[{"id":1,"player":0,"count":1},{"id":11,"player":0,"count":1}]}]},`+remoteVillager, "")
	p, enemy := g.Human(0), g.Human(1)
	before := g.Self(p).Units
	enemyBefore := g.Self(enemy).Units[0]
	g.Submit(p, harness.Order(defs.OrderType_BuildingResearch, []uint64{g.Self(p).Buildings[0].InternalID}, 1, false))
	g.EndTurn()
	g.EndTurn()
	if !g.Self(p).ResearchedTechs[1] {
		t.Fatal("farming did not finish")
	}
	for _, u := range before {
		if u.UnitID == 11 {
			after := g.State(p).Unit(u.InternalID)
			if after == nil || after.CombatStats != u.CombatStats || after.TurnStamina != u.TurnStamina {
				t.Error("farming affected nonmatching infantry")
			}
		}
	}
	if after := g.Self(enemy).Units[0]; after.CombatStats != enemyBefore.CombatStats || after.TurnStamina != enemyBefore.TurnStamina || g.Self(enemy).ResearchedTechs[1] {
		t.Error("farming affected the enemy")
	}
}
