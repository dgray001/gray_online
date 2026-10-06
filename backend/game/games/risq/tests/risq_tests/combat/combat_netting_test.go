package combat

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
	"math"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestTwoPhaseDeath(t *testing.T) {
	g := combatGame(t, 1, 1)
	p0, p1 := g.Human(0), g.Human(1)
	u0 := g.Self(p0).Units[0].InternalID
	u1 := g.Self(p1).Units[0].InternalID
	g.Submit(p0, harness.OrderAttackUnit([]uint64{u0}, u1))
	g.Submit(p1, harness.OrderAttackUnit([]uint64{u1}, u0))
	g.EndTurn()
	if len(g.Self(p0).Units) != 0 || len(g.Self(p1).Units) != 0 {
		t.Errorf("expected both units to die simultaneously, p0: %d, p1: %d", len(g.Self(p0).Units), len(g.Self(p1).Units))
	}
}

func TestSameTickDamageSums(t *testing.T) {
	single := combatGame(t, 11, 13)
	s0, s1 := single.Human(0), single.Human(1)
	su0, su1 := single.Self(s0).Units[0], single.Self(s1).Units[0]
	passive(single, s1, su1.InternalID)
	single.Submit(s0, harness.OrderAttackUnit([]uint64{su0.InternalID}, su1.InternalID))
	single.Submit(s1)
	one := su1.CombatStats.Health - single.Self(s1).Units[0].CombatStats.Health

	pair := combatGameTwoAttackers(t)
	p0, p1 := pair.Human(0), pair.Human(1)
	attackers := pair.Self(p0).Units
	target := pair.Self(p1).Units[0]
	passive(pair, p1, target.InternalID)
	pair.Submit(p0, harness.OrderAttackUnit([]uint64{attackers[0].InternalID, attackers[1].InternalID}, target.InternalID))
	pair.Submit(p1)
	two := target.CombatStats.Health - pair.Self(p1).Units[0].CombatStats.Health

	if math.Abs(two-2*one) > 1e-6 {
		t.Errorf("two attackers dealt %v, want twice a single attacker's %v", two, one)
	}
}

// Keeps a defender from retaliating, so attacker count is the only variable
func passive(g *harness.Game, human int, unit_id uint64) {
	g.Action(human, "set-unit-behavior", gin.H{"internal_ids": []uint64{unit_id}, "stance": uint8(defs.UnitStance_PASSIVE), "attack_back": false})
}
