package invariants

import "github.com/dgray001/gray_online/game/games/risq/tests/harness"

func verifyEviction(g *harness.Game, turn int) {
	g.T.Helper()
	for slot := range 2 {
		state := g.Self(g.Human(slot))
		garrisoned := 0
		for _, unit := range state.Units {
			if unit.GarrisonedIn != nil {
				garrisoned++
			}
		}
		if (turn == 0 || turn == 2) && garrisoned != 1 {
			g.T.Fatalf("garrison admitted %d units, want 1", garrisoned)
		}
		if turn >= 1 && len(state.Units) != 2 {
			g.T.Fatal("deleting a garrisoned unit left a ghost or removed another unit")
		}
		if turn >= 3 && (garrisoned != 0 || len(state.Buildings) != 0 || state.PopulationLimit != 0) {
			g.T.Fatal("building deletion did not release garrison and population support")
		}
	}
}
