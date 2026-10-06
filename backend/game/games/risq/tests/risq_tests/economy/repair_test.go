package economy

import (
	"math"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestRepairHealsProportionallyAndStopsAtFullHealth(t *testing.T) {
	g, p := damagedHousingGame(t)
	b := centerHousing(g, p)
	u := g.Self(p).Units[0]
	wood := g.Self(p).Resources.Wood
	wantHeal := 0.6 * float64(b.CombatStats.MaxHealth) / 14 * float64(u.CurrentStamina)
	g.Submit(p, harness.Order(defs.OrderType_UnitRepair, []uint64{u.InternalID}, int64(b.InternalID), false))
	g.EndTurn()
	after := centerHousing(g, p)
	healed := after.CombatStats.Health - b.CombatStats.Health
	if math.Abs(healed-wantHeal) > 0.0002 || after.CombatStats.Health >= float64(b.CombatStats.MaxHealth) {
		t.Errorf("first-turn healing %v, want %v and still damaged", healed, wantHeal)
	}
	wantCharge := (math.Round(after.CombatStats.Health*2500) - math.Round(b.CombatStats.Health*2500)) / 10000
	if got := wood - g.Self(p).Resources.Wood; math.Abs(got-wantCharge) > 0.00001 {
		t.Errorf("repair charged %v wood, want %v for the health restored", got, wantCharge)
	}
	for turn := 0; centerHousing(g, p).CombatStats.Health < float64(b.CombatStats.MaxHealth) && turn < 4; turn++ {
		g.EndTurn()
	}
	finished := centerHousing(g, p)
	if finished.InternalID != b.InternalID || finished.CombatStats.Health != float64(b.CombatStats.MaxHealth) {
		t.Errorf("repaired housing %+v, want same building at full health", finished)
	}
	wantTotal := (math.Round(float64(b.CombatStats.MaxHealth)*2500) - math.Round(b.CombatStats.Health*2500)) / 10000
	paid := wood - g.Self(p).Resources.Wood
	if math.Abs(paid-wantTotal) > 0.00001 {
		t.Errorf("total repair charge %v, want %v", paid, wantTotal)
	}
	g.EndTurn()
	if g.Self(p).Resources.Wood != wood-paid || centerHousing(g, p).CombatStats.Health != finished.CombatStats.Health {
		t.Error("completed repair kept charging or changed full health")
	}
}
