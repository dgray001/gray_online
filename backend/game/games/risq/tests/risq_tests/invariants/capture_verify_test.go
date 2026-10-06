package invariants

import "github.com/dgray001/gray_online/game/games/risq/tests/harness"

func verifyCapture(g *harness.Game, turn int) {
	g.T.Helper()
	x, want := 1, []int{0, 1, 0, -1}[turn]
	if turn == 0 {
		x = 0
	}
	owner := g.State(g.Human(0)).Space(x, 0).Ownership
	if owner == nil || *owner != want {
		g.T.Fatalf("capture turn %d space (%d,0): owner %v, want %d", turn, x, owner, want)
	}
	if turn == 1 && g.State(g.Human(0)).Unit(unitIDs(g, 0, 11)[0]).Space.X != 1 {
		g.T.Fatal("soldier did not enter enemy building's space")
	}
}
