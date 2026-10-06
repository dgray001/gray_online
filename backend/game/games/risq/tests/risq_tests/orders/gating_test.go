package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func TestSubmittedPlayersCannotChangeActions(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	rejectAction(g, p, "unsubmit-orders", nil, "Orders not submitted")
	g.Submit(p, harness.OrderMove(unitIDs(g, p, 1)[:1], 3, 0))
	g.EndTurn()
	g.Submit(p, harness.OrderMoveZone(unitIDs(g, p, 1)[:1], 0, 0, -1, 1))
	for kind, payload := range blockedActions(g, p) {
		rejectAction(g, p, kind, payload, "Orders already submitted")
	}
	g.Action(p, "unsubmit-orders", nil)
	g.Action(p, "set-unit-behavior", gin.H{"internal_ids": unitIDs(g, p, 11), "stance": 1})
	if g.Self(p).OrdersSubmitted {
		t.Fatal("unsubmit did not reopen actions")
	}
}
