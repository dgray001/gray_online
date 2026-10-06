package invariants

import "github.com/dgray001/gray_online/game/games/risq/tests/harness"

func verifyConstruction(g *harness.Game, turn int) {
	g.T.Helper()
	space := g.State(g.Human(0)).Space(0, 0)
	if len(space.Buildings) != 1 {
		g.T.Fatalf("contested foundation produced %d buildings", len(space.Buildings))
	}
	building := space.Buildings[0]
	if building.BuildingID != 2 || (turn >= 1 && building.UnderConstruction) {
		g.T.Fatal("winning housing did not complete")
	}
	for slot := range 2 {
		state := g.Self(g.Human(slot))
		want := 500.0
		if slot == building.PlayerID {
			want -= 30
		}
		if state.Resources.Wood != want || len(state.PlannedFoundations) != 0 {
			g.T.Errorf("player %d foundation refund/cleanup: wood %v, foundations %v", slot, state.Resources.Wood, state.PlannedFoundations)
		}
	}
}
