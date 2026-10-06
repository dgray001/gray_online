package orders

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func TestGeneratedOrderLifecycle(t *testing.T) {
	for _, typ := range []defs.OrderType{defs.OrderType_UnitMoveSpace, defs.OrderType_UnitMoveZone, defs.OrderType_UnitGather} {
		for _, replace := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/replace=%v", typ, replace), func(t *testing.T) {
				g := orderGame(t, richBank, "")
				p := g.Human(0)
				target := harness.ZoneKey(3, 0, 0, 0)
				point := gin.H{"location_kind": 2, "location_id": target}
				if typ == defs.OrderType_UnitMoveSpace {
					target = harness.SpaceKey(3, 0)
					point = gin.H{"location_kind": 1, "location_id": target}
				}
				if typ == defs.OrderType_UnitGather {
					target = harness.ZoneKey(0, 0, 1, -1)
					point = gin.H{"location_kind": 2, "location_id": target, "object_type": 3}
				}
				u := produceAtPoint(g, p, 1, 1, point)
				id := u.ActiveOrders[0].InternalID
				checkGeneratedOrder(g, p, u.InternalID, id, typ, target)
				g.EndTurn()
				checkGeneratedOrder(g, p, u.InternalID, id, typ, target)
				g.Submit(p, harness.OrderMoveZone([]uint64{u.InternalID}, 0, 0, -1, 1))
				if len(g.Self(p).ActiveOrders) != 2 {
					t.Fatal("pending order did not coexist with generated order")
				}
				g.Action(p, "unsubmit-orders", nil)
				checkGeneratedOrder(g, p, u.InternalID, id, typ, target)
				if len(g.Self(p).ActiveOrders) != 1 {
					t.Fatal("unsubmit did not preserve only generated order")
				}
				before, wood := *g.State(p).Unit(u.InternalID), g.Self(p).Resources.Wood
				order := harness.Order(defs.OrderType_CancelOrder, []uint64{u.InternalID}, int64(id), false)
				if replace {
					order = harness.Order(defs.OrderType_UnitMoveZone, []uint64{u.InternalID}, harness.ZoneKey(0, 0, -1, 1), true)
				}
				g.Submit(p, order)
				g.EndTurn()
				checkOrderGone(g, p, u.InternalID, id)
				if g.Self(p).Resources.Wood != wood {
					t.Fatal("cancelled or cleared gather order still credited resources")
				}
				if replace {
					queue := g.State(p).Unit(u.InternalID).ActiveOrders
					if len(queue) > 0 {
						checkGeneratedOrder(g, p, u.InternalID, queue[0].InternalID, defs.OrderType_UnitMoveZone, harness.ZoneKey(0, 0, -1, 1))
					}
					for range 4 {
						g.EndTurn()
					}
					current := g.State(p).Unit(u.InternalID)
					if current.Space != (harness.Coord{}) || current.Zone != (harness.Coord{X: -1, Y: 1}) || len(current.ActiveOrders) != 0 {
						t.Fatalf("generated order replacement did not finish: %+v", current)
					}
				} else {
					g.EndTurn()
					current := g.State(p).Unit(u.InternalID)
					if current.Space != before.Space || current.Zone != before.Zone || len(current.ActiveOrders) != 0 || g.Self(p).Resources.Wood != wood {
						t.Fatalf("cancelled generated order resumed: %+v", current)
					}
				}
				checkOrderGone(g, p, u.InternalID, id)
			})
		}
	}
}
