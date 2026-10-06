package shape

import "github.com/dgray001/gray_online/game/games/risq/tests/harness"

func unitID(g *harness.Game, slot, x int, kind uint32) uint64 {
	g.T.Helper()
	for _, unit := range g.Self(g.Human(slot)).Units {
		if unit.Space == (harness.Coord{X: x}) && unit.UnitID == kind {
			return unit.InternalID
		}
	}
	g.T.Fatalf("slot %d has no unit %d at (%d,0)", slot, kind, x)
	return 0
}

func centerID(g *harness.Game) uint64 {
	g.T.Helper()
	for _, building := range g.Self(g.Human(1)).Buildings {
		if building.BuildingID == 1 && building.Space == (harness.Coord{}) {
			return building.InternalID
		}
	}
	g.T.Fatal("target village center missing")
	return 0
}
