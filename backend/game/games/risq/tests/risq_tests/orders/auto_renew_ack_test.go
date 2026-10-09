package orders

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAutoRenewAcknowledgementIsOwnerOnly(t *testing.T) {
	g := orderGame(t, richBank, "")
	p, other := g.Human(0), g.Human(1)
	g.Action(p, "set-auto-renew", gin.H{"building_id": 3, "count": 0})
	rawAction(g, p, "set-auto-renew", gin.H{"building_id": 3, "count": 2})
	update := nextUpdate(t, g.Base.Players[uint64(p+1)].Updates)
	if update.Kind != "auto-renew-set" || update.Content["building_id"] != uint32(3) || update.Content["count"] != 2 || update.Content["game"] == nil || len(g.Self(p).ActiveOrders) != 0 {
		t.Fatalf("queue change did not acknowledge immediately: %+v", update)
	}
	if len(g.Base.Players[uint64(other+1)].Updates) != 0 || len(g.Base.ViewerUpdates) != 0 || len(g.Self(other).AutoRenewals) != 0 {
		t.Fatal("private auto-renew queue was broadcast")
	}
}
