package combat

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
	"testing"
)

func TestStanceAttackBack(t *testing.T) {
	g := combatGame(t, 13, 13)
	p0, p1 := g.Human(0), g.Human(1)
	u0 := g.Self(p0).Units[0]
	u1 := g.Self(p1).Units[0]

	// Set u1 to PASSIVE and attack_back to true
	stance := uint8(defs.UnitStance_PASSIVE)
	attackBack := true
	g.Action(p1, "set-unit-behavior", gin.H{"internal_ids": []uint64{u1.InternalID}, "stance": stance, "attack_back": attackBack})

	g.Submit(p0, harness.OrderAttackUnit([]uint64{u0.InternalID}, u1.InternalID))
	g.Submit(p1) // Idle, should attack back

	if g.Self(p0).Units[0].CombatStats.Health == u0.CombatStats.Health {
		t.Errorf("u1 did not attack back")
	}
}

func TestPassiveNoAttackBack(t *testing.T) {
	g := combatGame(t, 13, 13)
	p0, p1 := g.Human(0), g.Human(1)
	u0, u1 := g.Self(p0).Units[0], g.Self(p1).Units[0]
	stance := uint8(defs.UnitStance_PASSIVE)
	attackBack := false
	g.Action(p1, "set-unit-behavior", gin.H{"internal_ids": []uint64{u1.InternalID}, "stance": stance, "attack_back": attackBack})
	g.Submit(p0, harness.OrderAttackUnit([]uint64{u0.InternalID}, u1.InternalID))
	g.Submit(p1)
	if g.Self(p0).Units[0].CombatStats.Health != u0.CombatStats.Health {
		t.Errorf("u1 attacked back despite attack_back=false")
	}
}
