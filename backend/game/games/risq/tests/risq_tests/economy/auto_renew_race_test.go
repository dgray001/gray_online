package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func TestAutoRenewAllocatesSharedCreditByBuildingID(t *testing.T) {
	g := economyGame(t, `{"wood":90}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":3,"player":0},"units":[{"id":1,"player":0,"count":1}]},{"x":1,"y":0,"building":{"id":3,"player":0},"units":[{"id":1,"player":0,"count":1}]}]},`+remoteVillager, "")
	p := g.Human(0)
	config := defs.BuildingConfigs[3]
	config.Gather.Base_gather_speed = 250
	defs.BuildingConfigs[3] = config
	g.Action(p, "set-auto-renew", gin.H{"building_id": 3, "count": 1})
	orders := make([]defs.OrderFromFrontend, 0)
	winner := ^uint64(0)
	for _, b := range g.Self(p).Buildings {
		winner = min(winner, b.InternalID)
		for _, u := range g.Self(p).Units {
			if u.Zone == b.Zone && u.Space == b.Space {
				orders = append(orders, harness.OrderGather([]uint64{u.InternalID}, 0, 0, b.Zone.X, b.Zone.Y))
			}
		}
	}
	g.Submit(p, orders...)
	g.EndTurn()
	active := g.Self(p).ActiveOrders
	if len(active) != 1 || active[0].OrderType != uint8(defs.OrderType_UnitRenew) || active[0].TargetID != int64(winner) || g.Self(p).Resources.Wood != 30 || len(g.Self(p).AutoRenewals) != 0 {
		t.Fatalf("shared credit allocation was not deterministic: %+v", g.Self(p))
	}
}
