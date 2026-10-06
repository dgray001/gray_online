package orders

import (
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func checkOrderGone(g *harness.Game, p int, unit, id uint64) {
	g.T.Helper()
	for _, queue := range [][]gin.H{entity(g, p, "units", unit)["active_orders"].([]gin.H), playerPayload(g, p)["active_orders"].([]gin.H)} {
		for _, order := range queue {
			if order["internal_id"] == id {
				g.T.Fatalf("resolved generated order %d remained active", id)
			}
		}
	}
}
