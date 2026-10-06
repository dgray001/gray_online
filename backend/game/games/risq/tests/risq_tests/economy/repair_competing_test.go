package economy

import (
	"math"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestRepairsShareStockpileAcrossBuildings(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		g := economyGame(t, `{"wood":0.5}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":2,"player":0},"units":[{"id":1,"player":0,"count":1},{"id":1,"player":1,"count":1}]},{"x":1,"y":0,"building":{"id":2,"player":0},"units":[{"id":1,"player":0,"count":1},{"id":1,"player":1,"count":1}]}]}`, "")
		p, enemy := g.Human(0), g.Human(1)
		attacks := []defs.OrderFromFrontend{}
		for _, b := range g.Self(p).Buildings {
			for _, u := range g.Self(enemy).Units {
				if u.Zone == b.Zone {
					attacks = append(attacks, harness.Order(defs.OrderType_UnitAttackBuilding, []uint64{u.InternalID}, int64(b.InternalID), false))
				}
			}
		}
		g.Submit(enemy, attacks...)
		g.EndTurn()
		cancels := []defs.OrderFromFrontend{}
		for _, order := range g.Self(enemy).ActiveOrders {
			cancels = append(cancels, harness.Order(defs.OrderType_CancelOrder, nil, int64(order.InternalID), false))
		}
		g.Submit(enemy, cancels...)
		before := map[uint64]float64{}
		repairs := []defs.OrderFromFrontend{}
		for _, b := range g.Self(p).Buildings {
			before[b.InternalID] = b.CombatStats.Health
			if b.CombatStats.Health <= 0 || b.CombatStats.Health >= float64(b.CombatStats.MaxHealth)-1 {
				t.Fatal("attack did not leave enough damage for a constrained repair")
			}
			for _, u := range g.Self(p).Units {
				if u.Zone == b.Zone {
					repairs = append(repairs, harness.Order(defs.OrderType_UnitRepair, []uint64{u.InternalID}, int64(b.InternalID), false))
				}
			}
		}
		if len(attacks) != 2 || len(cancels) != 2 || len(repairs) != 2 {
			t.Fatal("setup did not produce two independent attacks and repairs")
		}
		if reverse {
			repairs[0], repairs[1] = repairs[1], repairs[0]
		}
		g.Submit(p, repairs...)
		for turn := 0; turn < 2; turn++ {
			state := g.Self(p)
			if state.Resources.Wood != 0 || len(state.ActiveOrders) != 0 || len(state.Refusals()) != 0 {
				t.Errorf("competing repairs retained funds or work: %+v, resources %+v", state, state.Resources)
			}
			for _, b := range state.Buildings {
				if gain := b.CombatStats.Health - before[b.InternalID]; math.Abs(gain-1) > 0.0001 {
					t.Errorf("reverse=%v, building %d healed %v, want 1 from its quarter-wood share", reverse, b.InternalID, gain)
				}
			}
			g.EndTurn()
		}
	}
}
