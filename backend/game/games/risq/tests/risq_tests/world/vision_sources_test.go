package world

import (
	"fmt"
	"testing"
)

func outpostAt(x, y int) string {
	return fmt.Sprintf(`{"x":%d,"y":%d,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":21,"player":0}}]}`, x, y)
}

func TestSpaceStaysVisibleWhileAnotherSourceCoversIt(t *testing.T) {
	spaces := spaceWith(0, 0, grass, unit(villager, 0)) + "," + spaceWith(1, 0, grass, "") + "," + outpostAt(2, 0) + "," +
		spaceWith(-1, 0, grass, "") + "," + spaceWith(-2, 0, grass, "") + "," + spaceWith(3, 0, grass, unit(villager, 1))
	g := startGame(t, mapDoc("", spaces))
	west := g.Human(0)
	if moveUntilArrived(g, west, -2, 0, 6) < 1 {
		t.Fatal("the villager never walked west")
	}
	state := g.State(west)
	if got := state.Space(1, 0).Visibility; got != good {
		t.Errorf("(1,0) visibility %d after the villager left, want good from the outpost", got)
	}
	if got := state.Space(0, 0).Visibility; got != poor {
		t.Errorf("(0,0) visibility %d, want poor from the outpost's second ring, not fog", got)
	}
}

func TestLosingTheLastUnitInASpaceKeepsPoorVisionForOneTurn(t *testing.T) {
	spaces := spaceWith(0, 0, grass, unit(villager, 0)+","+unit(infantry, 1)) + "," + spaceWith(0, 4, grass, unit(villager, 0))
	g := startGame(t, mapDoc("", spaces))
	west := g.Human(0)
	for turn := 0; turn < 5 && len(g.Self(west).Units) > 1; turn++ {
		g.EndTurn()
	}
	if units := len(g.Self(west).Units); units != 1 {
		t.Fatalf("setup: P0 has %d units, want the villager at (0,0) killed", units)
	}
	if got := g.State(west).Space(0, 0).Visibility; got != poor {
		t.Errorf("(0,0) visibility %d the turn its last unit died, want poor", got)
	}
	g.EndTurn()
	if got := g.State(west).Space(0, 0).Visibility; got != fog {
		t.Errorf("(0,0) visibility %d a turn later, want death vision expired to fog", got)
	}
}

// An outpost at (0,0) with no unit near it: P0's villager is far away at (0,4) and P1's at (3,0)
func TestBuildingsAloneProvideTheirConfiguredVision(t *testing.T) {
	spaces := outpostAt(0, 0) + "," + spaceWith(1, 0, grass, "") + "," + spaceWith(2, 0, grass, "") + "," +
		spaceWith(3, 0, grass, unit(villager, 1)) + "," + spaceWith(0, 4, grass, unit(villager, 0))
	g := startGame(t, mapDoc("", spaces))
	state := g.State(g.Human(0))
	for x, want := range []int{good, good, poor, unexplored} {
		if got := state.Space(x, 0).Visibility; got != want {
			t.Errorf("space (%d,0) visibility %d, want %d from the outpost's rings", x, got, want)
		}
	}
}
