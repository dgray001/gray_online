package combat

import (
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"math"
	"testing"
)

func TestAttackTypesAndPenetration(t *testing.T) {
	g := combatGame(t, 12, 13) // 12: 10 piercing att. 13: 3 piercing def, 50 hp
	p0, p1 := g.Human(0), g.Human(1)
	u0 := g.Self(p0).Units[0]
	u1 := g.Self(p1).Units[0]
	g.Submit(p0, harness.OrderAttackUnit([]uint64{u0.InternalID}, u1.InternalID))
	g.Submit(p1) // do nothing
	after1 := g.Self(p1).Units[0]
	damage := u1.CombatStats.Health - after1.CombatStats.Health
	// ratio 3/10 is under the knee: (10-3) * stamina / 10
	if want := 7 * float64(u0.CurrentStamina) / 10; math.Abs(damage-want) > 1e-6 {
		t.Errorf("expected damage %v, got %v", want, damage)
	}
}

func TestPenetrationReducesDefense(t *testing.T) {
	g := combatGame(t, 13, 13) // 18 attack, 5% penetration vs 4 blunt + 3 piercing defense
	p0, p1 := g.Human(0), g.Human(1)
	u0 := g.Self(p0).Units[0]
	u1 := g.Self(p1).Units[0]
	g.Submit(p0, harness.OrderAttackUnit([]uint64{u0.InternalID}, u1.InternalID))
	g.Submit(p1)
	damage := u1.CombatStats.Health - g.Self(p1).Units[0].CombatStats.Health
	defense := 4*0.95 + 3*0.95
	want := (18 - defense) * float64(u0.CurrentStamina) / 10
	if math.Abs(damage-want) > 1e-6 {
		t.Errorf("expected damage %v, got %v", want, damage)
	}
}
