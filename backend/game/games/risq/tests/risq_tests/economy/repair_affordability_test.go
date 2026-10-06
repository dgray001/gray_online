package economy

import (
	"math"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestRepairWorkersShareLimitedMixedResources(t *testing.T) {
	g, p := damagedRepairGame(t, `{"wood":0.2,"stone":0.1}`, 3)
	b := centerHousing(g, p)
	original := defs.BuildingConfigs[2]
	config := original
	config.Cost.Stone = 15
	defs.BuildingConfigs[2] = config
	t.Cleanup(func() { defs.BuildingConfigs[2] = original })
	ids := []uint64{}
	for _, u := range g.Self(p).Units {
		ids = append(ids, u.InternalID)
	}
	g.Submit(p, harness.Order(defs.OrderType_UnitRepair, ids, int64(b.InternalID), false))
	g.EndTurn()
	after := centerHousing(g, p)
	if gain := after.CombatStats.Health - b.CombatStats.Health; math.Abs(gain-0.8) > 0.001 || g.Self(p).Resources.Wood != 0 || g.Self(p).Resources.Stone != 0 {
		t.Fatalf("limited repair healed %v, resources %+v; want 0.8 health and exhausted funds", gain, g.Self(p).Resources)
	}
	g.EndTurn()
	if centerHousing(g, p).CombatStats.Health != after.CombatStats.Health || g.Self(p).Resources.Wood != 0 || g.Self(p).Resources.Stone != 0 || len(g.Self(p).ActiveOrders) != 0 {
		t.Error("repair continued after exhausting resources")
	}
}
