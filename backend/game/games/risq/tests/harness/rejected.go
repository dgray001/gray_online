package harness

import (
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
)

func (g *Game) SubmitRejected(i int, order defs.OrderFromFrontend, message string) {
	g.T.Helper()
	order.Player_id = g.PlayerID(i)
	g.Risq.PlayerAction(game.PlayerAction{Kind: "submit-orders", Client_id: int(g.clients[i]), Action: gin.H{"orders": []defs.OrderFromFrontend{order}}})
	if failed := g.Failed(i); len(failed) != 1 || failed[0] != message {
		g.T.Fatalf("submission failures %v, want %q", failed, message)
	}
	if g.Self(i).OrdersSubmitted {
		g.T.Fatal("rejected submission marked the player submitted")
	}
	g.drain()
}
