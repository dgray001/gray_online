package combat

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
	"math"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestTickHistoryIncomingDamageOnIdleUnit(t *testing.T) {
	t.Run("simultaneous-deaths-and-final-turn", func(t *testing.T) {
		g := combatGame(t, 1, 1)
		p, enemy := g.Human(0), g.Human(1)
		a, b := g.Self(p).Units[0], g.Self(enemy).Units[0]
		g.Submit(p, harness.OrderAttackUnit([]uint64{a.InternalID}, b.InternalID))
		g.Submit(enemy, harness.OrderAttackUnit([]uint64{b.InternalID}, a.InternalID))
		g.EndTurn()
		if !g.Base.GameEnded() || g.State(p).Unit(a.InternalID) != nil || g.State(enemy).Unit(b.InternalID) != nil {
			t.Fatal("fixture did not end with simultaneous deaths")
		}
		for _, u := range []harness.Unit{a, b} {
			replay := harness.RequireReplay(t, g.State(g.Human(u.PlayerID)))
			archived := replay.Unit(u.InternalID)
			if archived == nil || archived.UnitID != u.UnitID || archived.PlayerID != u.PlayerID {
				t.Fatalf("deleted actor identity missing: %+v", archived)
			}
			action := harness.RequireTickAction(t, archived.TickActions, "attack")
			if action.Execute.StaminaSpent <= 0 {
				t.Errorf("dying unit's attack was lost: %+v", action)
			}
			settled := replay.UnitAt(u.InternalID, replay.TickCount)
			if settled == nil || !settled.Deleted || settled.CombatStats.Health > 0 {
				t.Errorf("death absent from recorded state: %+v", settled)
			}
		}
	})
	g := combatGame(t, 11, 13)
	p, enemy := g.Human(0), g.Human(1)
	a, target := g.Self(p).Units[0], g.Self(enemy).Units[0]
	passive(g, enemy, target.InternalID)
	g.Submit(p, harness.OrderAttackUnit([]uint64{a.InternalID}, target.InternalID))
	g.EndTurn()
	after := g.State(enemy).Unit(target.InternalID)
	if after == nil || after.CombatStats.Health >= target.CombatStats.Health {
		t.Fatal("fixture did not damage an idle survivor")
	}
	harness.AssertTickSpend(t, after.TickActions, 0)
	harness.AssertTickEffect(t, after.TickActions, "damage", harness.TickTarget{Kind: "unit", InternalID: target.InternalID}, target.CombatStats.Health-after.CombatStats.Health)
	replay := harness.RequireReplay(t, g.State(enemy))
	settled := replay.UnitAt(target.InternalID, replay.TickCount)
	if settled == nil || settled.CombatStats.Health != after.CombatStats.Health {
		t.Errorf("settled idle unit health differs: %+v", settled)
	}
}

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
