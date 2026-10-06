package orders

import (
	"reflect"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func TestInvalidGatherPointsPreserveExistingPoint(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	id := buildingID(g, p, 1)
	g.Action(p, "set-gather-point", gin.H{"building_id": id, "location_kind": 1, "location_id": harness.SpaceKey(3, 0)})
	before := entity(g, p, "buildings", id)
	for _, payload := range []gin.H{
		{"location_kind": 0}, {"location_kind": 3}, {"location_kind": 255},
		{"location_kind": 1, "location_id": harness.SpaceKey(9, 0)},
		{"location_kind": 2, "location_id": harness.ZoneKey(0, 0, 9, 0)},
		{"location_kind": 2, "location_id": harness.ZoneKey(9, 0, 0, 0)},
		{"location_kind": 1, "object_type": 4}, {"location_kind": 1, "object_type": 255},
	} {
		payload["building_id"] = id
		g.Action(p, "set-gather-point", payload)
		if !reflect.DeepEqual(before, entity(g, p, "buildings", id)) {
			t.Fatalf("invalid point changed building: %v", payload)
		}
	}
	other, unsupported := buildingID(g, g.Human(1), 1), buildingID(g, p, 11)
	beforeOther, beforeUnsupported := entity(g, g.Human(1), "buildings", other), entity(g, p, "buildings", unsupported)
	for _, subject := range []uint64{999999, other, unsupported} {
		g.Action(p, "set-gather-point", gin.H{"building_id": subject, "location_kind": 1, "location_id": harness.SpaceKey(3, 0)})
		if !reflect.DeepEqual(before, entity(g, p, "buildings", id)) {
			t.Fatal("foreign or ineligible point changed owned building")
		}
		if !reflect.DeepEqual(beforeOther, entity(g, g.Human(1), "buildings", other)) || !reflect.DeepEqual(beforeUnsupported, entity(g, p, "buildings", unsupported)) {
			t.Fatal("gather point changed foreign or ineligible building")
		}
	}
}
