package invariants

import "github.com/dgray001/gray_online/game/games/risq/tests/harness"

func verifyProduction(g *harness.Game, turn int) {
	g.T.Helper()
	for slot := range 2 {
		state := g.Self(g.Human(slot))
		queued := 0
		for _, building := range state.Buildings {
			queued += len(building.ProductionQueue)
		}
		if state.Resources.Food != 400 {
			g.T.Fatal("production charged more than its two reservations")
		}
		if turn <= 1 && (len(state.Units) != 6 || queued != 1) {
			g.T.Fatalf("last-slot contest: population %d, queued %d", len(state.Units), queued)
		}
		if turn == 3 && (len(state.Units) != 6 || queued != 0) {
			g.T.Fatal("blocked production did not resume after deletion")
		}
	}
}
