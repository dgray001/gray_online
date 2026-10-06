package orders

import (
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func playerPayload(g *harness.Game, p int) gin.H {
	g.T.Helper()
	for _, player := range g.Risq.ToFrontend(uint64(p+1), false)["players"].([]gin.H) {
		if player["player"].(gin.H)["player_id"] == g.PlayerID(p) {
			return player
		}
	}
	g.T.Fatal("own player payload missing")
	return nil
}
