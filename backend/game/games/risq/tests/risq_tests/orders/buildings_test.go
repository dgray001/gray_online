package orders

import "github.com/dgray001/gray_online/game/games/risq/tests/harness"

func buildingID(g *harness.Game, p int, kind uint32) uint64 {
	g.T.Helper()
	for _, b := range g.Self(p).Buildings {
		if b.BuildingID == kind {
			return b.InternalID
		}
	}
	g.T.Fatalf("building %d missing", kind)
	return 0
}
