package invariants

import "github.com/dgray001/gray_online/game/games/risq/tests/harness"

func verifyBudget(g *harness.Game, turn int) {
	g.T.Helper()
	for slot := range 2 {
		state := g.Self(g.Human(slot))
		if state.Resources.Food != 0 || state.Resources.Wood != 0 || state.Resources.Stone != 0 {
			g.T.Fatalf("budget was not spent exactly once: %+v", state.Resources)
		}
		want := 2
		if turn == 2 {
			want = 1
		}
		if len(state.Units) != want {
			g.T.Fatalf("budget allowed %d units, want %d", len(state.Units), want)
		}
		if turn == 0 && len(state.Refusals()) != 1 {
			g.T.Fatal("insufficient-budget production was not refused")
		}
	}
}
