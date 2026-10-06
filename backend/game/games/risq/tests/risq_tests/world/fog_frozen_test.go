package world

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

// P0's villager at (0,0); P1's villager on a stone mine at (1,0) and P1's village center at (1,-1); open ground west of P0
func frozenGame(t *testing.T) (*harness.Game, int, int) {
	t.Helper()
	mine := `{"x":1,"y":0,"terrain":1,"zones":[{"x":1,"y":0,"resource":41,"units":[` + unit(villager, 1) + `]}]}`
	camp := `{"x":1,"y":-1,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":1}}]}`
	spaces := spaceWith(0, 0, grass, unit(villager, 0)) + "," + mine + "," + camp + "," +
		spaceWith(-1, 0, grass, "") + "," + spaceWith(-2, 0, grass, "") + "," + spaceWith(-3, 0, grass, "")
	g := startGame(t, mapDoc("", spaces))
	return g, g.Human(0), g.Human(1)
}

func TestFoggedResourcesStayFrozenWhileHidden(t *testing.T) {
	g, p0, p1 := frozenGame(t)
	g.Submit(p1, harness.OrderGather([]uint64{firstUnit(g, p1).InternalID}, 1, 0, 1, 0))
	if moveUntilArrived(g, p0, -3, 0, 12) < 1 {
		t.Fatal("P0 never walked away")
	}
	remembered := g.State(p0).Space(1, 0).Resources[0].ResourcesLeft
	g.EndTurn()
	g.EndTurn()

	p1Space := g.State(p1).Space(1, 0)
	p0Space := g.State(p0).Space(1, 0)

	var liveZoneRes, gotZoneRes float64
	for _, row := range p1Space.Zones {
		for _, z := range row {
			if z.Resource != nil {
				liveZoneRes = z.Resource.ResourcesLeft
			}
		}
	}
	for _, row := range p0Space.Zones {
		for _, z := range row {
			if z.Resource != nil {
				gotZoneRes = z.Resource.ResourcesLeft
			}
		}
	}

	if live := p1Space.Resources[0].ResourcesLeft; live >= remembered {
		t.Fatalf("setup: live mine holds %v, want less than the remembered %v", live, remembered)
	}
	if got := p0Space.Resources[0].ResourcesLeft; got != remembered {
		t.Errorf("fogged mine holds %v, want the remembered %v", got, remembered)
	}
	if liveZoneRes == 0 || liveZoneRes >= remembered {
		t.Fatalf("setup: live zone mine holds too much or is nil")
	}
	if gotZoneRes != remembered {
		t.Errorf("fogged zone mine holds wrong value, want remembered %v, got %v", remembered, gotZoneRes)
	}
}

func TestFoggedBuildingsStayFrozenUntilVisionReturns(t *testing.T) {
	farm := `{"x":1,"y":-1,"terrain":1,"zones":[{"x":1,"y":-1,"terrain_override":24,"building":{"id":3,"player":1}}]}`
	spaces := spaceWith(0, 0, grass, unit(villager, 0)) + "," + farm + "," +
		spaceWith(-1, 0, grass, "") + "," + spaceWith(-2, 0, grass, "") + "," + spaceWith(-3, 0, grass, "") + "," + spaceWith(2, -1, grass, unit(villager, 1))
	g := startGame(t, mapDoc("", spaces))
	p0, p1 := g.Human(0), g.Human(1)
	if moveUntilArrived(g, p0, -3, 0, 12) < 1 {
		t.Fatal("P0 never walked away")
	}
	camp := g.Self(p1).Buildings[0].InternalID
	g.Submit(p1, harness.Order(defs.OrderType_BuildingDelete, []uint64{camp}, 0, false))
	g.EndTurn()
	g.EndTurn()
	if live := g.Self(p1).Buildings; len(live) != 0 {
		t.Fatalf("setup: P1 still has %d buildings, want the camp deleted", len(live))
	}
	live := g.State(p1).Space(1, -1)
	if live.Visibility < poor {
		t.Fatal("P1 lost vision of the deleted farm")
	}
	for _, row := range live.Zones {
		for _, zone := range row {
			if zone.Building != nil || zone.TerrainOverride != 0 {
				t.Fatal("the deleted farm or its terrain override remains in the live zone")
			}
		}
	}

	p0Space := g.State(p0).Space(1, -1)
	if got := p0Space.Buildings; len(got) != 1 || got[0].InternalID != camp {
		t.Errorf("fogged buildings %v, want the remembered camp", got)
	}

	var gotZoneBld *uint64
	var gotTerrain uint32
	for _, row := range p0Space.Zones {
		for _, z := range row {
			if z.Building != nil {
				id := z.Building.InternalID
				gotZoneBld = &id
				gotTerrain = z.TerrainOverride
			}
		}
	}
	if gotZoneBld == nil || *gotZoneBld != camp {
		t.Errorf("fogged zone building missing or wrong id")
	}
	if gotTerrain != 24 {
		t.Errorf("fogged terrain override %d, want remembered terrain 24", gotTerrain)
	}

	if moveUntilArrived(g, p0, 0, 0, 12) < 1 {
		t.Fatal("P0 never returned to reveal the deleted farm")
	}
	p0Space = g.State(p0).Space(1, -1)
	if got := p0Space.Buildings; len(got) != 0 {
		t.Errorf("buildings %v after vision returned, want the stale camp gone", got)
	}

	var afterZoneBld *uint64
	var afterTerrain uint32
	for _, row := range p0Space.Zones {
		for _, z := range row {
			if z.Building != nil {
				id := z.Building.InternalID
				afterZoneBld = &id
			}
			if z.TerrainOverride != 0 {
				afterTerrain = z.TerrainOverride
			}
		}
	}
	if afterZoneBld != nil {
		t.Errorf("zone building %v after vision returned, want nil", *afterZoneBld)
	}
	if afterTerrain != 0 {
		t.Errorf("terrain override %v after vision returned, want 0", afterTerrain)
	}
}
