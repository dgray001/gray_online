package orders

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAutoRenewFailuresAreExplicitAndAtomic(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	for _, action := range []gin.H{{"building_id": 3}, {"building_id": 3, "count": -1}, {"building_id": 1, "count": 1}, {"building_id": 999, "count": 1}, {"building_id": 3, "count": 1000000}, {"building_id": 3, "count": "bad"}} {
		before := *g.Self(p).Resources
		rawAction(g, p, "set-auto-renew", action)
		update := nextUpdate(t, g.Base.Players[uint64(p+1)].FailedUpdates)
		if update.Kind != "set-auto-renew-failed" || *g.Self(p).Resources != before || len(g.Self(p).AutoRenewals) != 0 {
			t.Fatalf("invalid action mutated state or did not report failure: %+v", update)
		}
	}
}
