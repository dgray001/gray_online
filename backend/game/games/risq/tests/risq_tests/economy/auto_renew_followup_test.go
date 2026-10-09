package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func TestAutoRenewPreservesQueuedMovement(t *testing.T) {
	g := sourceGame(t, true)
	p := g.Human(0)
	g.Action(p, "set-auto-renew", gin.H{"building_id": 3, "count": 1})
	id := g.Self(p).Units[0].InternalID
	g.Submit(p, harness.OrderGather([]uint64{id}, 0, 0, 0, 0), harness.OrderMoveZone([]uint64{id}, 0, 0, 1, 0))
	for range 60 {
		g.EndTurn()
	}
	if g.Self(p).AutoRenewals[0].Count != 1 || farmZone(g, p).Building.ResourcesLeft != 0 || g.Self(p).Units[0].Zone.X != 1 {
		t.Fatalf("auto-renew displaced queued movement: %+v", g.Self(p))
	}
}
