package world

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestRegionsStayListedThroughFog(t *testing.T) {
	spaces := spaceWith(0, 0, grass, unit(villager, 0)) + "," + spaceWith(1, 0, grass, "") + "," + spaceWith(1, -1, grass, "") + "," +
		spaceWith(-1, 0, grass, "") + "," + spaceWith(-2, 0, grass, "") + "," + spaceWith(-3, 0, grass, "") + "," + spaceWith(-3, 3, grass, unit(villager, 1))
	g := startGame(t, mapDoc(`,"regions":[{"name":"Camp","gold_bonus":1,"spaces":[[1,0],[1,-1]]}]`, spaces))
	explorer := g.Human(0)
	if camp := regionNamed(g.State(explorer).Regions, "Camp"); camp == nil || len(camp.Spaces) != 2 {
		t.Fatalf("setup: Camp %+v, want both its spaces explored", camp)
	}
	if moveUntilArrived(g, explorer, -3, 0, 12) < 1 {
		t.Fatal("the unit never walked away")
	}
	state := g.State(explorer)
	if state.Space(1, 0).Visibility != fog {
		t.Fatalf("setup: (1,0) visibility %d, want fog", state.Space(1, 0).Visibility)
	}
	if camp := regionNamed(state.Regions, "Camp"); camp == nil || len(camp.Spaces) != 2 {
		t.Errorf("Camp %+v after walking away, want it still listed with both fogged spaces", camp)
	}
}

// What was last seen stays frozen: P1 takes (1,0) after P0 walks away, and P0's fogged view must not notice
func TestFoggedSpacesStayFrozenWhileHidden(t *testing.T) {
	spaces := spaceWith(0, 0, grass, unit(villager, 0)) + "," + spaceWith(1, 0, grass, "") + "," + spaceWith(2, 0, grass, "") + "," +
		spaceWith(3, 0, grass, unit(infantry, 1)) + "," + spaceWith(-1, 0, grass, "") + "," + spaceWith(-2, 0, grass, "") + "," + spaceWith(-3, 0, grass, "")
	g := startGame(t, mapDoc("", spaces))
	p0, p1 := g.Human(0), g.Human(1)
	if got := g.State(p0).Space(1, 0).Ownership; got == nil || *got != unowned {
		t.Fatalf("setup: (1,0) owner %v, want unowned", got)
	}
	if moveUntilArrived(g, p0, -3, 0, 12) < 1 {
		t.Fatal("P0 never walked away")
	}
	raider := firstUnit(g, p1).InternalID
	g.Submit(p1, harness.OrderMove([]uint64{raider}, 1, 0))
	for turn := 0; where(g, p1) != (harness.Coord{X: 1, Y: 0}); turn++ {
		if turn == 8 {
			t.Fatal("P1 never reached (1,0)")
		}
		g.EndTurn()
	}
	if live := g.State(p1).Space(1, 0).Ownership; live == nil || *live != g.PlayerID(p1) {
		t.Fatalf("setup: P1 sees (1,0) owned by %v, want itself", live)
	}
	space := g.State(p0).Space(1, 0)
	if space.Visibility != fog || space.Ownership == nil || *space.Ownership != unowned {
		t.Errorf("fogged space: visibility %d, owner %v; want fog and the remembered unowned", space.Visibility, space.Ownership)
	}
	for _, row := range space.Zones {
		for _, zone := range row {
			if zone.Ownership == nil || *zone.Ownership != unowned {
				t.Errorf("fogged zone %v owner %v, want the remembered unowned", zone.Coordinate, zone.Ownership)
			}
		}
	}
}

// P0's villager at (0,0) beside a stone mine at (1,0) and a P1 village center at (1,-1), with open ground west to (-3,0)
func TestFoggedSpacesRememberWhatWasSeen(t *testing.T) {
	mine := `{"x":1,"y":0,"terrain":1,"zones":[{"x":1,"y":0,"resource":41}]}`
	camp := `{"x":1,"y":-1,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":1}}]}`
	spaces := spaceWith(0, 0, grass, unit(villager, 0)) + "," + mine + "," + camp + "," + spaceWith(-1, 0, grass, "") + "," +
		spaceWith(-2, 0, grass, "") + "," + spaceWith(-3, 0, grass, "") + "," + spaceWith(-3, 3, grass, unit(villager, 1))
	g := startGame(t, mapDoc("", spaces))
	explorer := g.Human(0)
	before := g.State(explorer).Space(1, 0)
	if before.Visibility != poor || len(before.Resources) != 1 {
		t.Fatalf("setup: (1,0) visibility %d with %d resources, want poor with 1", before.Visibility, len(before.Resources))
	}
	if turns := moveUntilArrived(g, explorer, -3, 0, 12); turns < 1 {
		t.Fatal("the unit never walked away")
	}
	state := g.State(explorer)
	for _, c := range [][2]int{{1, 0}, {1, -1}} {
		if v := state.Space(c[0], c[1]).Visibility; v != fog {
			t.Fatalf("(%d,%d) visibility %d, want fog after walking away", c[0], c[1], v)
		}
	}
	if got := state.Space(1, 0).Resources; len(got) != 1 || got[0].ResourceID != 41 {
		t.Errorf("fogged (1,0) resources %v, want the remembered stone mine", got)
	}
	if got := state.Space(1, -1).Buildings; len(got) != 1 || got[0].BuildingID != 1 {
		t.Errorf("fogged (1,-1) buildings %v, want the remembered village center", got)
	}
}
