package orders

import (
	"slices"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func produceAtPoint(g *harness.Game, p int, producer, kind uint32, point gin.H) harness.Unit {
	g.T.Helper()
	old := unitIDs(g, p, kind)
	point["building_id"] = buildingID(g, p, producer)
	g.Action(p, "set-gather-point", point)
	g.Submit(p, harness.OrderProduce([]uint64{buildingID(g, p, producer)}, kind))
	for range 4 {
		g.EndTurn()
		for _, u := range g.Self(p).Units {
			if u.UnitID == kind && !slices.Contains(old, u.InternalID) {
				return u
			}
		}
	}
	g.T.Fatal("no unit produced within four turns")
	return harness.Unit{}
}
