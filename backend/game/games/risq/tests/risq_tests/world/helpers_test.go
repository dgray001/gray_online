package world

import (
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

// The human's only unit; fails the test when they have none or several, since unit order in the payload is arbitrary
func firstUnit(g *harness.Game, human int) harness.Unit {
	g.T.Helper()
	units := g.Self(human).Units
	if len(units) != 1 {
		g.T.Fatalf("human %d has %d units, want exactly 1", human, len(units))
	}
	return units[0]
}

// Where the human's only unit stands; a garrisoned unit has no position, so that fails the test
func where(g *harness.Game, human int) harness.Coord {
	g.T.Helper()
	unit := firstUnit(g, human)
	if unit.GarrisonedIn != nil {
		g.T.Fatalf("human %d's unit is garrisoned and has no position", human)
	}
	return unit.Space
}

// The id of the human's unit of this kind standing in the space
func unitAt(g *harness.Game, human int, unitID uint32, x, y int) uint64 {
	g.T.Helper()
	for _, u := range g.Self(human).Units {
		if u.UnitID == unitID && u.Space == (harness.Coord{X: x, Y: y}) {
			return u.InternalID
		}
	}
	g.T.Fatalf("human %d has no unit %d at (%d,%d)", human, unitID, x, y)
	return 0
}

// Orders the human's only unit to a space, then ends turns until it arrives or the limit passes; returns the turns used, -1 if it never did
func moveUntilArrived(g *harness.Game, human int, x, y int, limit int) int {
	g.T.Helper()
	g.Submit(human, harness.OrderMove([]uint64{firstUnit(g, human).InternalID}, x, y))
	for turn := 1; turn <= limit; turn++ {
		g.EndTurn()
		if where(g, human) == (harness.Coord{X: x, Y: y}) {
			return turn
		}
	}
	return -1
}
