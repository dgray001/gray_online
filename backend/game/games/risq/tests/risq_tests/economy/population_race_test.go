package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestSimultaneousProductionRespectsLastPopulationSlot(t *testing.T) {
	g := productionGame(t, 9, 2, "")
	p := g.Human(0)
	buildings := centers(g, p)
	winner := buildings[0].InternalID
	if buildings[1].InternalID < winner {
		winner = buildings[1].InternalID
	}
	g.Submit(p, harness.OrderProduce([]uint64{buildings[0].InternalID}, 1), harness.OrderProduce([]uint64{buildings[1].InternalID}, 1))
	for turn := 0; turn < 2; turn++ {
		g.EndTurn()
		state := g.Self(p)
		queued := len(centers(g, p)[0].ProductionQueue) + len(centers(g, p)[1].ProductionQueue)
		if len(state.Units) != 10 || state.PopulationLimit != 10 || queued != 1 || state.Resources.Food != 200 {
			t.Errorf("turn %d: units %d, cap %d, queued %d, food %v; want 10, 10, 1, 200", turn, len(state.Units), state.PopulationLimit, queued, state.Resources.Food)
		}
		for _, b := range centers(g, p) {
			if (len(b.ProductionQueue) == 0) != (b.InternalID == winner) {
				t.Errorf("building %d won=%v, want winner %d", b.InternalID, len(b.ProductionQueue) == 0, winner)
			}
			for _, item := range b.ProductionQueue {
				if item.ItemID != 1 || item.StaminaRemaining <= 0 {
					t.Errorf("pending production %+v, want unfinished villager", item)
				}
			}
		}
	}
}
