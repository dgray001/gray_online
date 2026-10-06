package combat

import (
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"math"
	"testing"
)

func TestHealthNetting(t *testing.T) {
	g := combatGame(t, 13, 13)
	p0, p1 := g.Human(0), g.Human(1)
	u0 := g.Self(p0).Units[0]
	u1 := g.Self(p1).Units[0]
	g.Submit(p0, harness.OrderAttackUnit([]uint64{u0.InternalID}, u1.InternalID))
	g.Submit(p1, harness.OrderAttackUnit([]uint64{u1.InternalID}, u0.InternalID))
	after0 := g.Self(p0).Units[0]
	after1 := g.Self(p1).Units[0]
	damage0 := u0.CombatStats.Health - after0.CombatStats.Health
	damage1 := u1.CombatStats.Health - after1.CombatStats.Health
	if damage0 <= 0 || math.Abs(damage0-damage1) > 1e-6 {
		t.Errorf("expected equal simultaneous damage, got p0: %v, p1: %v", damage0, damage1)
	}
}
