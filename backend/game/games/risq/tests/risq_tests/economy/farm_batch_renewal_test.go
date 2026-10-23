package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestRenewalBatchCannotOverspend(t *testing.T) {
	g := economyGame(t, `{"wood":60}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":203,"player":0},"units":[{"id":11,"player":0,"count":1}]},{"x":1,"y":0,"building":{"id":203,"player":0},"units":[{"id":11,"player":0,"count":1}]}]},`+remoteWorker, "")
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
	original := defs.BuildingConfigs[203]
	config := original
	config.Gather.Base_gather_speed = 250
	defs.BuildingConfigs[203] = config
	defer func() { defs.BuildingConfigs[203] = original }()
	g.Submit(p, gather...)
	for turn := 0; turn < 60; turn++ {
		g.EndTurn()
		orchards := g.Self(p).Buildings
		if orchards[0].ResourcesLeft == 0 && orchards[1].ResourcesLeft == 0 {
			break
		}
	}
	defs.BuildingConfigs[203] = original
	for _, b := range g.Self(p).Buildings {
		if b.ResourcesLeft != 0 {
			t.Fatal("setup did not deplete both orchards")
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
	if state.Resources.Wood != 20 || len(state.ActiveOrders) != 1 || state.ActiveOrders[0].OrderType != uint8(defs.OrderType_UnitGather) || len(state.Refusals()) != 1 || state.Refusals()[0] != "cannot afford renew" {
		t.Fatalf("renewal batch overspent or retained unpaid work: %+v, resources %+v", state, state.Resources)
	}
	for turn := 0; turn < 2; turn++ {
		for _, b := range g.Self(p).Buildings {
			if int64(b.InternalID) == renew[0].Target_id && b.ResourcesLeft <= 0 {
				t.Errorf("orchard %d did not refill, wood %v", b.InternalID, g.Self(p).Resources.Wood)
			}
			if int64(b.InternalID) != renew[0].Target_id && b.ResourcesLeft != 0 {
				t.Errorf("unpaid orchard %d refilled to %v", b.InternalID, b.ResourcesLeft)
			}
			if g.Self(p).Resources.Wood != 20 {
				t.Errorf("orchard %d changed wood balance to %v", b.InternalID, g.Self(p).Resources.Wood)
			}
		}
		g.EndTurn()
	}
}
