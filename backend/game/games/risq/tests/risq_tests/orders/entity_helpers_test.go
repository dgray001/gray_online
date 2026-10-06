package orders

import (
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func entity(g *harness.Game, p int, collection string, id uint64) gin.H {
	g.T.Helper()
	for _, player := range g.Risq.ToFrontend(uint64(p+1), false)["players"].([]gin.H) {
		if player["player"].(gin.H)["player_id"] == g.PlayerID(p) {
			for _, item := range player[collection].([]gin.H) {
				if item["internal_id"] == id {
					return item
				}
			}
		}
	}
	g.T.Fatalf("%s %d missing", collection, id)
	return nil
}
