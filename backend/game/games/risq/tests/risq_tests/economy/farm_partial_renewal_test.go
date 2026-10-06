package economy

import (
	"math"
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
		if want := 200 * float64(stamina) / 48; pool >= 200 || math.Abs(pool-want) > 0.001 || g.Self(p).Resources.Wood != 60 {
			t.Fatalf("%d renewers: pool %v, wood %v; want partial refill %v and one payment", workers, pool, g.Self(p).Resources.Wood, want)
		}
		for _, u := range g.Self(p).Units {
			if u.CurrentStamina != u.TurnStamina {
				t.Errorf("renewal retained unspent stamina: %+v", u)
			}
		}
		for turn := 0; turn < 6 && farmZone(g, p).Building.ResourcesLeft < 200; turn++ {
			g.EndTurn()
		}
		g.EndTurn()
		if farmZone(g, p).Building.ResourcesLeft != 200 || farmZone(g, p).Building.InternalID != id || g.Self(p).Resources.Wood != 60 || len(g.Self(p).ActiveOrders) != 0 {
			t.Errorf("%d renewers failed to finish once: %+v", workers, g.Self(p))
		}
	}
}
