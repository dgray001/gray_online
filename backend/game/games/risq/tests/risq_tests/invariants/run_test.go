package invariants

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

func runScenario(t *testing.T, scenario scenario, permutation []int) []string {
	t.Helper()
	g := scenario.setup(t)
	trajectory := []string{canonical(normalize("", jsonValue(t, checkState(t, g, scenario.players, true))))}
	for turn := 0; turn < scenario.turns; turn++ {
		batches := make([][]defs.OrderFromFrontend, scenario.players)
		for slot := range batches {
			batches[slot] = scenario.orders(g, turn, slot)
		}
		before := g.State(g.Human(0)).TurnNumber
		for _, slot := range permutation {
			g.Submit(g.Human(slot), batches[slot]...)
		}
		if after := g.State(g.Human(0)).TurnNumber; after != before+1 {
			t.Fatalf("turn advanced from %d to %d", before, after)
		}
		scenario.verify(g, turn)
		views := checkState(t, g, scenario.players, scenario.capLossTurn == 0 || turn+1 < scenario.capLossTurn)
		trajectory = append(trajectory, canonical(normalize("", jsonValue(t, views))))
	}
	return trajectory
}
