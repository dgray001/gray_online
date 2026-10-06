package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestProductionQueueSpawnsOncePerOrder(t *testing.T) {
	g := productionGame(t, 1, 1, "")
	p := g.Human(0)
	b := centers(g, p)[0]
	order := harness.OrderProduce([]uint64{b.InternalID}, 1)
	g.Submit(p, order, order)
	for turn := 1; turn <= 3; turn++ {
		g.EndTurn()
		state := g.Self(p)
		wantUnits, wantQueue := 3, 0
		if turn == 1 {
			wantUnits, wantQueue = 2, 1
		}
		if len(state.Units) != wantUnits || len(centers(g, p)[0].ProductionQueue) != wantQueue || state.Resources.Food != 200 {
			t.Errorf("turn %d: units %d, queue %v, food %v; want %d units, %d queued, 200 food", turn, len(state.Units), centers(g, p)[0].ProductionQueue, state.Resources.Food, wantUnits, wantQueue)
		}
		for _, u := range state.Units {
			if u.UnitID != 1 || u.Space != b.Space || u.Zone != b.Zone {
				t.Errorf("unexpected produced unit %+v", u)
			}
		}
	}
}
