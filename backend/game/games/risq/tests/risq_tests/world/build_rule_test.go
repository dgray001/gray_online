package world

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

const housing = 2

// P0 owns (0,0) with a soldier and has a villager there; P1's soldier owns (1,0); (-1,0) and (-3,0) are unowned
func buildGame(t *testing.T) (*harness.Game, int) {
	t.Helper()
	spaces := spaceWith(0, 0, grass, unit(infantry, 0)+","+unit(villager, 0)) + "," + spaceWith(1, 0, grass, unit(infantry, 1)) + "," +
		spaceWith(-1, 0, grass, "") + "," + spaceWith(-2, 0, grass, "") + "," + spaceWith(-3, 0, grass, "")
	g := startGame(t, mapDoc("", spaces))
	if wood := g.Self(g.Human(0)).Resources.Wood; wood != 200 {
		t.Fatalf("setup: wood %v, want the default 200", wood)
	}
	return g, g.Human(0)
}

func TestCannotBuildInAnEnemyOwnedSpace(t *testing.T) {
	g, p0 := buildGame(t)
	g.Submit(p0, harness.OrderBuild([]uint64{unitAt(g, p0, villager, 0, 0)}, housing, 1, 0, 1, 0))
	g.EndTurn()
	if wood := g.Self(p0).Resources.Wood; wood != 200 {
		t.Errorf("wood %v, want 200: the order was refused so nothing should be spent", wood)
	}
	if refusals := g.Self(p0).Refusals(); len(refusals) != 1 || refusals[0] != "not receivable" {
		t.Errorf("refusals %v, want the one build order reported as not receivable", refusals)
	}
}

func TestBuildingInAnUnownedNeighborSucceeds(t *testing.T) {
	g, p0 := buildGame(t)
	g.Submit(p0, harness.OrderBuild([]uint64{unitAt(g, p0, villager, 0, 0)}, housing, -1, 0, 1, 0))
	for turn := 0; turn < 6 && len(g.Self(p0).Buildings) == 0; turn++ {
		g.EndTurn()
	}
	buildings := g.Self(p0).Buildings
	if len(buildings) != 1 || buildings[0].Space != (harness.Coord{X: -1, Y: 0}) {
		t.Errorf("buildings %v, want one housing at (-1,0)", buildings)
	}
}

func TestBuildingInAnUnownedSpaceFarFromOwnedLandNeverStarts(t *testing.T) {
	g, p0 := buildGame(t)
	builder := unitAt(g, p0, villager, 0, 0)
	g.Submit(p0, harness.OrderBuild([]uint64{builder}, housing, -3, 0, 1, 0))
	for turn := 0; turn < 8; turn++ {
		g.EndTurn()
	}
	if buildings := g.Self(p0).Buildings; len(buildings) != 0 {
		t.Errorf("buildings %v, want none: (-3,0) does not border owned land", buildings)
	}
	if at := g.State(p0).Unit(builder).Space; at != (harness.Coord{X: -3, Y: 0}) {
		t.Errorf("the villager is at %v, want it to have walked to (-3,0) and been refused there", at)
	}
}
