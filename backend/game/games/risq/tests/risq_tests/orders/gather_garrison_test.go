package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func TestBuildingGatherPointGarrisonsNewUnit(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	target := buildingID(g, p, 1)
	u := produceAtPoint(g, p, 1, 1, gin.H{"location_kind": 2, "location_id": harness.ZoneKey(0, 0, 0, 0), "object_type": 2, "object_id": target})
	if u.GarrisonedIn == nil || *u.GarrisonedIn != target {
		t.Fatalf("new unit was not garrisoned: %+v", u)
	}
}
