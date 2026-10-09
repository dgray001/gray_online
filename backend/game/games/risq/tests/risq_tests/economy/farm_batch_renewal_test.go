package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestRenewalBatchCannotOverspend(t *testing.T) {
	g := economyGame(t, `{"wood":90}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":3,"player":0},"units":[{"id":1,"player":0,"count":1}]},{"x":1,"y":0,"building":{"id":3,"player":0},"units":[{"id":1,"player":0,"count":1}]}]},`+remoteVillager, "")
	p := g.Human(0)
	buildings := g.Self(p).Buildings
	gather, renew := []defs.OrderFromFrontend{}, []defs.OrderFromFrontend{}
	for _, b := range buildings {
		for _, u := range g.Self(p).Units {
			if u.Zone == b.Zone {
				gather = append(gather, harness.OrderGather([]uint64{u.InternalID}, 0, 0, b.Zone.X, b.Zone.Y))
			}
		}
	}
	original := defs.BuildingConfigs[3]
	config := original
	config.Gather.Base_gather_speed = 250
	defs.BuildingConfigs[3] = config
	defer func() { defs.BuildingConfigs[3] = original }()
	g.Submit(p, gather...)
	for turn := 0; turn < 60; turn++ {
		g.EndTurn()
		farms := g.Self(p).Buildings
		if farms[0].ResourcesLeft == 0 && farms[1].ResourcesLeft == 0 {
			break
		}
	}
	defs.BuildingConfigs[3] = original
	for _, b := range g.Self(p).Buildings {
		if b.ResourcesLeft != 0 {
			t.Fatal("setup did not deplete both farms")
		}
		for _, u := range g.Self(p).Units {
			if u.Zone == b.Zone {
				renew = append(renew, harness.Order(defs.OrderType_UnitRenew, []uint64{u.InternalID}, int64(b.InternalID), false))
			}
		}
	}
	g.Submit(p, renew...)
	g.EndTurn()
	state := g.Self(p)
	if state.Resources.Wood != 30 || len(state.ActiveOrders) != 1 || state.ActiveOrders[0].OrderType != uint8(defs.OrderType_UnitGather) || len(state.Refusals()) != 1 || state.Refusals()[0] != "cannot afford renew" {
		t.Fatalf("renewal batch overspent or retained unpaid work: %+v, resources %+v", state, state.Resources)
	}
	for turn := 0; turn < 2; turn++ {
		for _, b := range g.Self(p).Buildings {
			if int64(b.InternalID) == renew[0].Target_id && b.ResourcesLeft <= 0 {
				t.Errorf("farm %d did not refill, wood %v", b.InternalID, g.Self(p).Resources.Wood)
			}
			if int64(b.InternalID) != renew[0].Target_id && b.ResourcesLeft != 0 {
				t.Errorf("unpaid farm %d refilled to %v", b.InternalID, b.ResourcesLeft)
			}
			if g.Self(p).Resources.Wood != 30 {
				t.Errorf("farm %d changed wood balance to %v", b.InternalID, g.Self(p).Resources.Wood)
			}
		}
		g.EndTurn()
	}
}
