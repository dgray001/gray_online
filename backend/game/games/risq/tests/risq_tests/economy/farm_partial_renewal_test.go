package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestPartialRenewalConsumesStaminaAndSharesPayment(t *testing.T) {
	for _, workers := range []int{1, 2} {
		g := farmGame(t, 120, workers)
		p := g.Human(0)
		id := fastExhaustFarm(g, p)
		original := defs.BuildingConfigs[3]
		config := original
		config.Gather.Renew_stamina = 48
		defs.BuildingConfigs[3] = config
		t.Cleanup(func() { defs.BuildingConfigs[3] = original })
		ids, stamina := []uint64{}, 0
		for _, u := range g.Self(p).Units {
			ids = append(ids, u.InternalID)
			stamina += u.CurrentStamina
		}
		g.Submit(p, harness.Order(defs.OrderType_UnitRenew, ids, int64(id), false))
		g.EndTurn()
		pool := farmZone(g, p).Building.ResourcesLeft
		if pool != 0 || g.Self(p).Resources.Wood != 60 || farmZone(g, p).Building.RenewStaminaRemaining != 48-stamina {
			t.Fatalf("%d renewers: pool %v, remaining %d, wood %v; want no refill and one payment", workers, pool, farmZone(g, p).Building.RenewStaminaRemaining, g.Self(p).Resources.Wood)
		}
		for _, u := range g.Self(p).Units {
			if u.CurrentStamina != u.TurnStamina {
				t.Errorf("renewal retained unspent stamina: %+v", u)
			}
		}
		for turn := 0; turn < 6 && farmZone(g, p).Building.ResourcesLeft < 200; turn++ {
			g.EndTurn()
		}
		if pool := farmZone(g, p).Building.ResourcesLeft; pool <= 0 || farmZone(g, p).Building.InternalID != id || g.Self(p).Resources.Wood != 60 || len(g.Self(p).ActiveOrders) != workers {
			t.Errorf("%d renewers failed to finish once: %+v", workers, g.Self(p))
		}
		for _, u := range g.Self(p).Units {
			if len(u.ActiveOrders) != 1 || u.ActiveOrders[0].OrderType != uint8(defs.OrderType_UnitGather) {
				t.Errorf("renewing villager did not gather: %+v", u.ActiveOrders)
			}
		}
	}
}
