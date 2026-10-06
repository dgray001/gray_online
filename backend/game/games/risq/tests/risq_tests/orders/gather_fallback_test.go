package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func TestMilitaryResourceGatherPointFallsBackToMovement(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	before := g.State(p).Space(0, 0).Resources[0].ResourcesLeft
	u := produceAtPoint(g, p, 22, 11, gin.H{"location_kind": 2, "location_id": harness.ZoneKey(0, 0, 1, -1), "object_type": 3})
	if u.Space != (harness.Coord{}) || u.Zone != (harness.Coord{X: 1, Y: -1}) || len(u.ActiveOrders) != 0 {
		t.Fatalf("fallback move did not finish: %+v", u)
	}
	if g.State(p).Space(0, 0).Resources[0].ResourcesLeft != before {
		t.Fatal("military gather point gathered resources")
	}
}
