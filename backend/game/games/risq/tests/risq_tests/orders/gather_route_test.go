package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func TestNewUnitsReceiveGatherPointOrders(t *testing.T) {
	for _, c := range []struct {
		name  string
		point gin.H
		order defs.OrderType
	}{
		{"space", gin.H{"location_kind": 1, "location_id": harness.SpaceKey(3, 0)}, defs.OrderType_UnitMoveSpace},
		{"zone", gin.H{"location_kind": 2, "location_id": harness.ZoneKey(3, 0, 0, 0)}, defs.OrderType_UnitMoveZone},
		{"resource", gin.H{"location_kind": 2, "location_id": harness.ZoneKey(0, 0, 1, -1), "object_type": 3}, defs.OrderType_UnitGather},
		{"missing unit", gin.H{"location_kind": 2, "location_id": harness.ZoneKey(3, 0, 0, 0), "object_type": 1, "object_id": 999999}, defs.OrderType_UnitMoveZone},
		{"missing building", gin.H{"location_kind": 2, "location_id": harness.ZoneKey(3, 0, 0, 0), "object_type": 2, "object_id": 999999}, defs.OrderType_UnitMoveZone},
		{"space ignores object", gin.H{"location_kind": 1, "location_id": harness.SpaceKey(3, 0), "object_type": 3}, defs.OrderType_UnitMoveSpace},
	} {
		t.Run(c.name, func(t *testing.T) {
			g := orderGame(t, richBank, "")
			p := g.Human(0)
			u := produceAtPoint(g, p, 1, 1, c.point)
			if len(u.ActiveOrders) != 1 || u.ActiveOrders[0].OrderType != uint8(c.order) || u.ActiveOrders[0].InternalID == 0 {
				t.Fatalf("wrong synthetic order: %+v", u.ActiveOrders)
			}
			if c.order == defs.OrderType_UnitGather && g.Self(p).Resources.Wood <= 1000 {
				t.Fatal("gather-point unit did not gather")
			}
		})
	}
}
