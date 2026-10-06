package world

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestLosingARegionStopsItsBonus(t *testing.T) {
	g, p0, p1 := incomeGame(t, spaceIncome+eastWest, 0)
	g.Submit(p1, harness.OrderMove([]uint64{firstUnit(g, p1).InternalID}, 1, 0))
	var before float64
	for turn := 0; turn < 6 && regionNamed(g.State(p0).Regions, "West").Owner == g.PlayerID(p0); turn++ {
		before = g.Self(p0).Resources.Gold
		g.EndTurn()
	}
	if owner := regionNamed(g.State(p0).Regions, "West").Owner; owner != unowned {
		t.Fatalf("setup: West owner %d, want it contested and unowned", owner)
	}
	if got := g.Self(p0).Resources.Gold - before; got != 3 {
		t.Errorf("P0 earned %v after losing West, want 3 (one owned space, no bonus)", got)
	}
}
