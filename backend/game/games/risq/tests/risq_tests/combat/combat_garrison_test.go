package combat

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"math"
	"testing"
)

func TestGarrisonSwings(t *testing.T) {
	g := garrisonCombatGame(t)
	p0, p1 := g.Human(0), g.Human(1)
	u0 := g.Self(p0).Units[0]
	u1 := g.Self(p1).Units[0]
	b0 := g.Self(p0).Buildings[0]

	// Garrison u0 in b0
	g.Submit(p0, harness.OrderGarrison([]uint64{u0.InternalID}, b0.InternalID))
	g.Submit(p1) // p1 does nothing

	// Now u0 is garrisoned. Order building to attack u1.
	before := g.Self(p1).Units[0].CombatStats.Health
	g.Submit(p0, harness.Order(defs.OrderType_BuildingAttackUnit, []uint64{b0.InternalID}, int64(u1.InternalID), false))
	g.Submit(p1) // u1 does nothing

	damage := before - g.Self(p1).Units[0].CombatStats.Health
	// Building 21 has 0 attack, but max garrison piercing 3. u0 has 9 piercing.
	// So capped at 3 piercing. u1 (id 13) has 3 piercing defense. 3 - 3 = 0.
	// But damage is clamped to positive due to floor.
	if damage <= 0 || math.IsNaN(damage) {
		t.Errorf("expected positive damage from garrison attack, got %v", damage)
	}
}
