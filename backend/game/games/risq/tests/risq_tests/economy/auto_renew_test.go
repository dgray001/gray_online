package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func TestAutoRenewCreditsRestoreRepeatedGathering(t *testing.T) {
	g := sourceGame(t, true)
	speedUpGathering(t)
	p := g.Human(0)
	g.Action(p, "set-auto-renew", gin.H{"building_id": 3, "count": 2})
	if g.Self(p).Resources.Wood != 0 || g.Self(p).AutoRenewals[0].Count != 2 {
		t.Fatal("renewal credits were not prepaid")
	}
	submitGather(g, p)
	for range 5 {
		g.EndTurn()
	}
	if g.Self(p).Resources.Food != 600 || g.Self(p).Resources.Wood != 0 || len(g.Self(p).AutoRenewals) != 0 {
		t.Fatalf("renewals did not yield three farm pools without recharging: %+v", g.Self(p))
	}
}

func TestAutoRenewQueueRefundsStoredPrices(t *testing.T) {
	g := sourceGame(t, true)
	p := g.Human(0)
	g.Action(p, "set-auto-renew", gin.H{"building_id": 3, "count": 1})
	original := defs.BuildingConfigs[3]
	config := original
	config.Gather.Renew_cost.Wood = 30
	defs.BuildingConfigs[3] = config
	t.Cleanup(func() { defs.BuildingConfigs[3] = original })
	g.Invalidate()
	g.Risq.PlayerAction(game.PlayerAction{Kind: "set-auto-renew", Client_id: p + 1, Action: gin.H{"building_id": 3, "count": 4}})
	if len(g.Failed(p)) != 1 || g.Self(p).Resources.Wood != 60 || g.Self(p).AutoRenewals[0].Count != 1 {
		t.Fatal("unaffordable increase partially changed the queue")
	}
	g.Action(p, "set-auto-renew", gin.H{"building_id": 3, "count": 2})
	if g.Self(p).Resources.Wood != 30 || g.Self(p).AutoRenewals[0].Count != 2 {
		t.Fatal("new credit did not use the changed price")
	}
	g.Action(p, "set-auto-renew", gin.H{"building_id": 3, "count": 1})
	if g.Self(p).Resources.Wood != 60 {
		t.Fatal("newest credit was not refunded at its paid price")
	}
	g.Action(p, "set-auto-renew", gin.H{"building_id": 3, "count": 0})
	if g.Self(p).Resources.Wood != 120 || len(g.Self(p).AutoRenewals) != 0 {
		t.Fatal("original credit was not refunded at its paid price")
	}
}

func TestAutoRenewPreservesQueuedMovement(t *testing.T) {
	g := sourceGame(t, true)
	speedUpGathering(t)
	p := g.Human(0)
	g.Action(p, "set-auto-renew", gin.H{"building_id": 3, "count": 1})
	id := g.Self(p).Units[0].InternalID
	g.Submit(p, harness.OrderGather([]uint64{id}, 0, 0, 0, 0), harness.OrderMoveZone([]uint64{id}, 0, 0, 1, 0))
	for range 6 {
		g.EndTurn()
	}
	if g.Self(p).AutoRenewals[0].Count != 1 || farmZone(g, p).Building.ResourcesLeft != 0 || g.Self(p).Units[0].Zone.X != 1 {
		t.Fatalf("auto-renew displaced queued movement: %+v", g.Self(p))
	}
}

func TestAutoRenewAllocatesSharedCreditByBuildingID(t *testing.T) {
	g := economyGame(t, `{"wood":90}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":3,"player":0},"units":[{"id":1,"player":0,"count":1}]},{"x":1,"y":0,"building":{"id":3,"player":0},"units":[{"id":1,"player":0,"count":1}]}]},`+remoteVillager, "")
	p := g.Human(0)
	original := defs.BuildingConfigs[3]
	config := original
	config.Gather.Base_gather_speed = 250
	defs.BuildingConfigs[3] = config
	t.Cleanup(func() { defs.BuildingConfigs[3] = original })
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
