package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func TestGatherPointSetReplaceAndClear(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	id := buildingID(g, p, 1)
	g.Action(p, "set-gather-point", gin.H{"building_id": id, "location_kind": 1, "location_id": harness.SpaceKey(3, 0)})
	gp := entity(g, p, "buildings", id)["gather_point"].(gin.H)
	if gp["location_kind"] != risq.RisqGatherPointLocationKind_SPACE || gp["location_id"] != uint64(harness.SpaceKey(3, 0)) {
		t.Fatalf("space gather point: %v", gp)
	}
	zone := harness.ZoneKey(0, 0, 1, -1)
	g.Action(p, "set-gather-point", gin.H{"building_id": id, "location_kind": 2, "location_id": zone, "object_type": 3, "object_id": 11})
	gp = entity(g, p, "buildings", id)["gather_point"].(gin.H)
	if gp["location_kind"] != risq.RisqGatherPointLocationKind_ZONE || gp["location_id"] != uint64(zone) || gp["object_type"] != risq.RisqGatherObjectType_RESOURCE || gp["object_id"] != uint64(11) {
		t.Fatalf("zone gather point: %v", gp)
	}
	g.Action(p, "set-gather-point", gin.H{"building_id": id, "clear": true})
	if _, present := entity(g, p, "buildings", id)["gather_point"]; present {
		t.Fatal("gather point survived clear")
	}
	if g.Self(p).OrdersSubmitted || len(g.Self(p).ActiveOrders) != 0 {
		t.Fatal("gather point entered order pipeline")
	}
}
