package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func TestUngarrisonGatherPointHandoff(t *testing.T) {
	for _, mode := range []string{"gather", "self", "queued-move", "automatic-move", "no-point"} {
		t.Run(mode, func(t *testing.T) {
			g := orderGame(t, richBank, "")
			p := g.Human(0)
			building := buildingID(g, p, 1)
			u := produceAtPoint(g, p, 1, 1, gin.H{"location_kind": 2, "location_id": harness.ZoneKey(0, 0, 0, 0), "object_type": 2, "object_id": building})
			if mode == "no-point" {
				g.Action(p, "set-gather-point", gin.H{"building_id": building, "clear": true})
			} else if mode != "self" {
				g.Action(p, "set-gather-point", gin.H{"building_id": building, "location_kind": 2, "location_id": harness.ZoneKey(0, 0, 1, -1), "object_type": 3})
			}
			wood := g.Self(p).Resources.Wood
			orders := []defs.OrderFromFrontend{harness.Order(defs.OrderType_UnitUngarrison, []uint64{u.InternalID}, 0, false)}
			move := harness.Order(defs.OrderType_UnitMoveZone, []uint64{u.InternalID}, harness.ZoneKey(0, 0, -1, 1), false)
			if mode == "queued-move" {
				orders = append(orders, move)
			} else if mode == "automatic-move" {
				orders = []defs.OrderFromFrontend{move}
			}
			g.Submit(p, orders...)
			g.EndTurn()
			u = *g.State(p).Unit(u.InternalID)
			if u.GarrisonedIn != nil {
				t.Fatal("unit was re-garrisoned")
			}
			if mode == "gather" {
				if g.Self(p).Resources.Wood <= wood || len(u.ActiveOrders) != 1 || u.ActiveOrders[0].OrderType != uint8(defs.OrderType_UnitGather) {
					t.Fatalf("ungarrisoned unit did not gather: %+v", u)
				}
			} else if len(u.ActiveOrders) != 0 || g.Self(p).Resources.Wood != wood {
				t.Fatalf("ungarrison overrode existing behavior: %+v", u)
			}
			if (mode == "queued-move" || mode == "automatic-move") && u.Zone != (harness.Coord{X: -1, Y: 1}) {
				t.Fatalf("existing move was not preserved: %+v", u)
			}
		})
	}
}
