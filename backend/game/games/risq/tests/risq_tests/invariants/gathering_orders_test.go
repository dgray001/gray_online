package invariants

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func gatheringOrders(g *harness.Game, turn, slot int) []defs.OrderFromFrontend {
	if turn != 0 {
		return nil
	}
	ids := unitIDs(g, slot, 1)
	return []defs.OrderFromFrontend{harness.OrderGather(ids[1:], 0, 0, 0, 0)}
}

func verifyGathering(g *harness.Game, turn int) {
	g.T.Helper()
	a, b := g.Self(g.Human(0)), g.Self(g.Human(1))
	if a.Resources.Food+b.Resources.Food != 1003.5 || a.Resources.Food != b.Resources.Food {
		g.T.Fatalf("contested gathering banks %v/%v, want equal shares totaling 1003.5", a.Resources.Food, b.Resources.Food)
	}
	if len(g.State(g.Human(0)).Space(0, 0).Resources) != 0 {
		g.T.Fatal("exhausted node survived")
	}
}
