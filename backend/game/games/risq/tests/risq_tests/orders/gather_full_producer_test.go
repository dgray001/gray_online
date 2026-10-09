package orders

import (
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
	"testing"
)

func TestFullProducerGatherPointSpawnsOutside(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	target := buildingID(g, p, 1)
	g.Submit(p, harness.OrderGarrison(unitIDs(g, p, 1), target))
	g.EndTurn()
	for range 3 {
		produceAtPoint(g, p, 1, 1, gin.H{"location_kind": 2, "location_id": harness.ZoneKey(0, 0, 0, 0), "object_type": 2, "object_id": target})
	}
	u := produceAtPoint(g, p, 1, 1, gin.H{"location_kind": 2, "location_id": harness.ZoneKey(0, 0, 0, 0), "object_type": 2, "object_id": target})
	if u.GarrisonedIn != nil || u.Space != (harness.Coord{}) || u.Zone != (harness.Coord{}) {
		t.Fatalf("full producer did not spawn outside: %+v", u)
	}
}
