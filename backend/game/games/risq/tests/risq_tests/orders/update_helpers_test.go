package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func rawAction(g *harness.Game, p int, kind string, action gin.H) {
	g.Risq.PlayerAction(game.PlayerAction{Client_id: p + 1, Kind: kind, Action: action})
}

func nextUpdate(t *testing.T, updates <-chan *game.UpdateMessage) *game.UpdateMessage {
	t.Helper()
	select {
	case u := <-updates:
		return u
	default:
		t.Fatal("missing update")
		return nil
	}
}

func latestUpdate(t *testing.T, updates <-chan *game.UpdateMessage) *game.UpdateMessage {
	t.Helper()
	u := nextUpdate(t, updates)
	for len(updates) > 0 {
		u = nextUpdate(t, updates)
	}
	return u
}
