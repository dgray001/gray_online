package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
)

func TestAutoRenewCreditsRestoreRepeatedGathering(t *testing.T) {
	g := sourceGame(t, true)
	p := g.Human(0)
	g.Action(p, "set-auto-renew", gin.H{"building_id": 3, "count": 2})
	if g.Self(p).Resources.Wood != 0 || g.Self(p).AutoRenewals[0].Count != 2 {
		t.Fatal("renewal credits were not prepaid")
	}
	submitGather(g, p)
	for range 120 {
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
	config := defs.BuildingConfigs[3]
	config.Gather.Renew_cost.Wood = 30
	defs.BuildingConfigs[3] = config
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
