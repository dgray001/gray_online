package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

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
