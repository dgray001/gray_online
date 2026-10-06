package orders

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func checkGeneratedOrder(g *harness.Game, p int, unit, id uint64, typ defs.OrderType, target int64) {
	g.T.Helper()
	if id == 0 {
		g.T.Fatal("generated order has no received ID")
	}
	queue := entity(g, p, "units", unit)["active_orders"].([]gin.H)
	if len(queue) != 1 || queue[0]["internal_id"] != id || queue[0]["order_type"] != typ || queue[0]["target_id"] != target {
		g.T.Fatalf("generated unit order mismatch: %v", queue)
	}
	for _, order := range playerPayload(g, p)["active_orders"].([]gin.H) {
		if order["internal_id"] == id {
			if snapshotJSON(g.T, order) != snapshotJSON(g.T, queue[0]) {
				g.T.Fatal("player and unit disagree on generated order")
			}
			return
		}
	}
	g.T.Fatalf("generated order %d missing from player", id)
}
