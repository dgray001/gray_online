package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func TestClearedGatherPointLeavesNewUnitIdle(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	g.Action(p, "set-gather-point", gin.H{"building_id": buildingID(g, p, 1), "location_kind": 1, "location_id": harness.SpaceKey(3, 0)})
	u := produceAtPoint(g, p, 1, 1, gin.H{"clear": true})
	if len(u.ActiveOrders) != 0 || u.Space != (harness.Coord{}) || u.Zone != (harness.Coord{}) {
		t.Fatalf("cleared point still routed unit: %+v", u)
	}
}
