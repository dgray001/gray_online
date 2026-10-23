package world

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

const (
	shelter = 202
	wall    = 211
)

// P0 owns (0,0) with a soldier and has a worker there; P1's soldier owns (1,0); (-1,0) and (-3,0) are unowned
func buildGame(t *testing.T) (*harness.Game, int) {
	t.Helper()
	spaces := spaceWith(0, 0, flatTerrain, unit(grunt, 0)+","+unit(worker, 0)) + "," + spaceWith(1, 0, flatTerrain, unit(grunt, 1)) + "," +
		spaceWith(-1, 0, flatTerrain, "") + "," + spaceWith(-2, 0, flatTerrain, "") + "," + spaceWith(-3, 0, flatTerrain, "")
	g := startGame(t, mapDoc("", spaces))
	if wood := g.Self(g.Human(0)).Resources.Wood; wood != 200 {
		t.Fatalf("setup: wood %v, want the default 200", wood)
	}
	return g, g.Human(0)
}

func TestCannotBuildInAnEnemyOwnedSpace(t *testing.T) {
	g, p0 := buildGame(t)
	g.Submit(p0, harness.OrderBuild([]uint64{unitAt(g, p0, worker, 0, 0)}, shelter, 1, 0, 1, 0))
	g.EndTurn()
	if wood := g.Self(p0).Resources.Wood; wood != 200 {
		t.Errorf("wood %v, want 200: the order was refused so nothing should be spent", wood)
	}
	if refusals := g.Self(p0).Refusals(); len(refusals) != 1 || refusals[0] != "not receivable" {
		t.Errorf("refusals %v, want the one build order reported as not receivable", refusals)
	}
}

func TestWallsAreEdgeZoneOnly(t *testing.T) {
	t.Run("center", func(t *testing.T) {
		g, p0 := buildGame(t)
		order := harness.OrderBuild([]uint64{unitAt(g, p0, worker, 0, 0)}, wall, 0, 0, 0, 0)
		g.SubmitRejected(p0, order, "Building id 211 can only be built on edge zones")
	})
	t.Run("edge", func(t *testing.T) {
		g, p0 := buildGame(t)
		g.Submit(p0, harness.OrderBuild([]uint64{unitAt(g, p0, worker, 0, 0)}, wall, 0, 0, -1, 0))
		for turn := 0; turn < 6 && len(g.Self(p0).Buildings) == 0; turn++ {
			g.EndTurn()
		}
		buildings := g.Self(p0).Buildings
		if len(buildings) != 1 || buildings[0].BuildingID != wall || buildings[0].Zone != (harness.Coord{X: -1, Y: 0}) {
			t.Errorf("buildings %v, want one wall on edge zone (-1,0)", buildings)
		}
	})
}

func TestBuildingInAnUnownedNeighborSucceeds(t *testing.T) {
	g, p0 := buildGame(t)
	g.Submit(p0, harness.OrderBuild([]uint64{unitAt(g, p0, worker, 0, 0)}, shelter, -1, 0, 1, 0))
	for turn := 0; turn < 6 && len(g.Self(p0).Buildings) == 0; turn++ {
		g.EndTurn()
	}
	buildings := g.Self(p0).Buildings
	if len(buildings) != 1 || buildings[0].Space != (harness.Coord{X: -1, Y: 0}) {
		t.Errorf("buildings %v, want one shelter at (-1,0)", buildings)
	}
}

func TestBuildingInAnUnownedSpaceFarFromOwnedLandNeverStarts(t *testing.T) {
	g, p0 := buildGame(t)
	builder := unitAt(g, p0, worker, 0, 0)
	g.Submit(p0, harness.OrderBuild([]uint64{builder}, shelter, -3, 0, 1, 0))
	for turn := 0; turn < 8; turn++ {
		g.EndTurn()
	}
	if buildings := g.Self(p0).Buildings; len(buildings) != 0 {
		t.Errorf("buildings %v, want none: (-3,0) does not border owned land", buildings)
	}
	if at := g.State(p0).Unit(builder).Space; at != (harness.Coord{X: -3, Y: 0}) {
		t.Errorf("the worker is at %v, want it to have walked to (-3,0) and been refused there", at)
	}
}
