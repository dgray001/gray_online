package economy

import (
	"math"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestRepairWorkersClampSharedHealingAtFullHealth(t *testing.T) {
	g, p := damagedRepairGame(t, `{"wood":120}`, 3)
	b := centerHousing(g, p)
	ids := []uint64{}
	for _, u := range g.Self(p).Units {
		ids = append(ids, u.InternalID)
	}
	wood := g.Self(p).Resources.Wood
	g.Submit(p, harness.Order(defs.OrderType_UnitRepair, ids, int64(b.InternalID), false))
	for turn := 0; centerHousing(g, p).CombatStats.Health < float64(b.CombatStats.MaxHealth) && turn < 4; turn++ {
		g.EndTurn()
	}
	finished := centerHousing(g, p)
	if finished.CombatStats.Health != float64(b.CombatStats.MaxHealth) {
		t.Errorf("health = %v, want max %v", finished.CombatStats.Health, b.CombatStats.MaxHealth)
	}
	wantTotal := (math.Round(float64(b.CombatStats.MaxHealth)*2500) - math.Round(b.CombatStats.Health*2500)) / 10000
	paid := wood - g.Self(p).Resources.Wood
	if math.Abs(paid-wantTotal) > 0.00001 {
		t.Errorf("total repair charge %v, want %v", paid, wantTotal)
	}
	g.EndTurn()
	if g.Self(p).Resources.Wood != wood-paid {
		t.Error("completed repair kept charging")
	}
}
