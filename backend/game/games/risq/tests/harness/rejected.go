package harness

import (
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
)

func (g *Game) SubmitRejected(i int, order defs.OrderFromFrontend, message string) {
	g.T.Helper()
	g.Invalidate()
	order.Player_id = g.PlayerID(i)
	before := g.Risq.StateHash()
	g.Risq.PlayerAction(game.PlayerAction{Kind: "submit-orders", Client_id: int(g.clients[i]), Action: gin.H{"orders": []defs.OrderFromFrontend{order}}})
	if failed := g.Failed(i); len(failed) != 1 || failed[0] != message {
		g.T.Fatalf("submission failures %v, want %q", failed, message)
	}
	if g.Risq.StateHash() != before {
		g.T.Fatal("rejected submission changed game state")
	}
	g.drain()
}
