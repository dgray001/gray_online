package world

import (
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

const unowned = -1

// P0 is present in every space, so its view lists every owner; slot numbers equal engine player ids
func ownershipGame(t *testing.T) (*harness.Game, int) {
	t.Helper()
	camp := `{"x":0,"y":-1,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":0},"units":[` + unit(infantry, 1) + `]}]}`
	spaces := []string{
		spaceWith(0, 0, grass, unit(infantry, 0)),
		spaceWith(1, 0, grass, unit(villager, 0)),
		spaceWith(-1, 0, grass, unit(infantry, 0)+","+unit(infantry, 1)),
		camp,
		spaceWith(2, 0, grass, ""),
	}
	g := startGame(t, mapDoc("", strings.Join(spaces, ",")))
	return g, g.Human(0)
}

func TestMilitaryUnitsClaimAndEconomicUnitsDoNot(t *testing.T) {
	g, p0 := ownershipGame(t)
	state := g.State(p0)
	for _, c := range []struct {
		x, y, owner int
		why         string
	}{
		{0, 0, 0, "a sole military unit claims its space"},
		{1, 0, unowned, "villagers never conquer"},
		{2, 0, unowned, "an empty space is nobody's"},
	} {
		if got := state.Space(c.x, c.y).Ownership; got == nil || *got != c.owner {
			t.Errorf("(%d,%d) owner %v, want %d: %s", c.x, c.y, got, c.owner, c.why)
		}
	}
}

func TestContestedSpacesAreUnownedAndBuildingsOutrankUnits(t *testing.T) {
	g, p0 := ownershipGame(t)
	state := g.State(p0)
	if got := state.Space(-1, 0).Ownership; got == nil || *got != unowned {
		t.Errorf("a space with both players' soldiers has owner %v, want none", got)
	}
	if got := state.Space(0, -1).Ownership; got == nil || *got != 0 {
		t.Errorf("a space with P0's building and P1's soldier has owner %v, want P0 (buildings decide first)", got)
	}
}

func TestZonesInheritTheirSpacesOwner(t *testing.T) {
	g, p0 := ownershipGame(t)
	for _, row := range g.State(p0).Space(0, 0).Zones {
		for _, zone := range row {
			if zone.Ownership == nil || *zone.Ownership != 0 {
				t.Errorf("zone %v owner %v, want 0 like its space", zone.Coordinate, zone.Ownership)
			}
		}
	}
}

func TestOwnershipFollowsUnitsAsTheyMove(t *testing.T) {
	g, p0 := ownershipGame(t)
	soldier := unitAt(g, p0, infantry, 0, 0)
	g.Submit(p0, harness.OrderMove([]uint64{soldier}, 2, 0))
	for turn := 0; turn < 8 && g.State(p0).Unit(soldier).Space != (harness.Coord{X: 2, Y: 0}); turn++ {
		g.EndTurn()
	}
	state := g.State(p0)
	if got := state.Space(2, 0).Ownership; got == nil || *got != 0 {
		t.Errorf("(2,0) owner %v after the soldier arrived, want 0", got)
	}
	if got := state.Space(0, 0).Ownership; got == nil || *got != unowned {
		t.Errorf("(0,0) owner %v after the soldier left, want it released", got)
	}
}
