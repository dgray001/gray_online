package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestPopulationCapPausesAndResumesProduction(t *testing.T) {
	g := productionGame(t, 5, 1, "")
	p := g.Human(0)
	g.Submit(p, harness.OrderProduce([]uint64{centers(g, p)[0].InternalID}, 1))
	g.EndTurn()
	queue := centers(g, p)[0].ProductionQueue
	if state := g.Self(p); len(state.Units) != 5 || state.PopulationLimit != 5 || state.Resources.Food != 250 || len(queue) != 1 || queue[0].StaminaRemaining != 10 {
		t.Fatalf("capped production: units %d, cap %d, food %v, queue %v", len(state.Units), state.PopulationLimit, state.Resources.Food, queue)
	}
	deleted := g.Self(p).Units[0].InternalID
	g.Submit(p, harness.Order(defs.OrderType_UnitDelete, []uint64{deleted}, 0, false))
	for turn := 0; turn < 2; turn++ {
		g.EndTurn()
		if state := g.Self(p); len(state.Units) > state.PopulationLimit {
			t.Fatal("production exceeded the population cap")
		}
	}
	state := g.Self(p)
	if len(state.Units) != 5 || state.Resources.Food != 250 || len(centers(g, p)[0].ProductionQueue) != 0 || g.State(p).Unit(deleted) != nil {
		t.Errorf("resumed production: units %d, food %v, queue %v; want replacement at cap with no new charge", len(state.Units), state.Resources.Food, centers(g, p)[0].ProductionQueue)
	}
}
