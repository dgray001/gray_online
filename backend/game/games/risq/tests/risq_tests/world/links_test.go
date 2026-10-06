package world

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

// Two spaces six steps apart, P0's villager on the west one, joined only by the given connections
func farApartGame(t *testing.T, connections string) (*harness.Game, int) {
	t.Helper()
	spaces := spaceWith(-3, 0, grass, unit(villager, 0)) + "," + spaceWith(3, 0, grass, unit(villager, 1))
	g := startGame(t, mapDoc(connections, spaces))
	return g, g.Human(0)
}

func TestUnitsCrossASeamThroughItsEdgeZonesBothWays(t *testing.T) {
	cases := map[string]struct {
		fromSlot int
		to, zone harness.Coord
	}{
		"west to east": {0, harness.Coord{X: 3, Y: 0}, harness.Coord{X: 1, Y: 0}},
		"east to west": {1, harness.Coord{X: -3, Y: 0}, harness.Coord{X: -1, Y: 0}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			g, _ := farApartGame(t, `,"connections":[{"from":[-3,0],"to":[3,0],"direction":3}]`)
			mover := g.Human(c.fromSlot)
			if turns := moveUntilArrived(g, mover, c.to.X, c.to.Y, 3); turns != 1 {
				t.Errorf("arrived after %d turns, want 1", turns)
			}
			if zone := firstUnit(g, mover).Zone; zone != c.zone {
				t.Errorf("unit entered at zone %v, want the facing edge %v", zone, c.zone)
			}
		})
	}
}

func TestSpacesWithoutAConnectionStayUnreachable(t *testing.T) {
	g, mover := farApartGame(t, "")
	if moveUntilArrived(g, mover, 3, 0, 4) != -1 || where(g, mover) != (harness.Coord{X: -3, Y: 0}) {
		t.Errorf("unit at %v, want it stuck on its own space", where(g, mover))
	}
}

func TestGarrisonedUnitsLeaveTheBuildingToMove(t *testing.T) {
	west := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":22,"player":0},"units":[` + unit(villager, 0) + `]}]}`
	g := startGame(t, mapDoc("", west+","+spaceWith(1, 0, grass, unit(villager, 1))))
	mover := g.Human(0)
	barracks, villagerID := g.Self(mover).Buildings[0].InternalID, firstUnit(g, mover).InternalID
	g.Submit(mover, harness.OrderGarrison([]uint64{villagerID}, barracks))
	g.EndTurn()
	if firstUnit(g, mover).GarrisonedIn == nil {
		t.Fatal("the unit did not garrison")
	}
	if turns := moveUntilArrived(g, mover, 1, 0, 3); turns != 1 {
		t.Errorf("arrived after %d turns, want 1 (ungarrison, then the border crossing)", turns)
	}
	if firstUnit(g, mover).GarrisonedIn != nil {
		t.Error("the unit is still garrisoned after leaving")
	}
}
