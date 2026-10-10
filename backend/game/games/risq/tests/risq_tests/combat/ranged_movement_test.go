package combat

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
	"testing"
)

func TestTickHistoryAreaAttackTargets(t *testing.T) {
	t.Run("shared-order-per-unit-targets", func(t *testing.T) {
		g := meetingGame(t, spaceOf(0, 0, flat, placed{heavy, 0, 0, 0}, placed{heavy, 0, 1, 0}, placed{heavy, 1, 0, 0}, placed{heavy, 1, 1, 0}))
		p, enemy := g.Human(0), g.Human(1)
		ids, targets := []uint64{}, map[harness.Coord]uint64{}
		for _, u := range g.Self(p).Units {
			ids = append(ids, u.InternalID)
			passive(g, p, u.InternalID)
		}
		for _, u := range g.Self(enemy).Units {
			targets[u.Zone] = u.InternalID
			passive(g, enemy, u.InternalID)
		}
		g.Submit(p, harness.Order(defs.OrderType_UnitAttackSpace, ids, harness.SpaceKey(0, 0), false))
		g.EndTurn()
		for _, u := range g.Self(p).Units {
			a := harness.RequireTickAction(t, u.TickActions, "attack")
			if a.Execute.Target.InternalID != targets[u.Zone] || a.Order.OrderType != uint8(defs.OrderType_UnitAttackSpace) {
				t.Errorf("shared order lost per-unit target: %+v", a)
			}
		}
	})
	t.Run("retarget-after-kill", func(t *testing.T) {
		g := retargetGame(t)
		p, enemy := g.Human(0), g.Human(1)
		u := g.Self(p).Units[0]
		passive(g, p, u.InternalID)
		g.Submit(p, harness.Order(defs.OrderType_UnitAttackSpace, []uint64{u.InternalID}, harness.SpaceKey(0, 0), false))
		g.EndTurn()
		if len(g.Self(enemy).Units) != 1 {
			t.Fatal("fixture did not kill exactly one target")
		}
		targets := map[uint64]bool{}
		for _, a := range g.State(p).Unit(u.InternalID).TickActions {
			if a.Intent.Kind != "attack" || a.Execute.Outcome != "executed" {
				continue
			}
			targets[a.Execute.Target.InternalID] = true
			if a.Execute.Target.Kind != "unit" || a.Execute.TargetLocation != (harness.TickLocation{Zone: harness.Coord{X: 1}}) {
				t.Errorf("historical target location lost: %+v", a)
			}
		}
		if len(targets) != 2 {
			t.Errorf("recorded %d targets, want both including the dead unit", len(targets))
		}
	})
	t.Run("synthetic-stance-order", func(t *testing.T) {
		g := stanceGame(t, sameZone)
		p, enemy := g.Human(0), g.Human(1)
		u, target := g.Self(p).Units[0], g.Self(enemy).Units[0]
		passive(g, enemy, target.InternalID)
		g.EndTurn()
		if g.Self(enemy).Units[0].CombatStats.Health >= target.CombatStats.Health {
			t.Fatal("fixture did not attack by stance")
		}
		a := harness.RequireTickAction(t, g.State(p).Unit(u.InternalID).TickActions, "attack")
		if a.Order.Source != "stance" || a.Order.OrderType != uint8(defs.OrderType_UnitAutoAttackUnit) || a.Execute.Target.InternalID != target.InternalID {
			t.Errorf("synthetic order provenance missing: %+v", a)
		}
		if orders := g.Self(p).ActiveOrders; len(orders) != 1 || orders[0].OrderType != uint8(defs.OrderType_UnitAutoAttackUnit) {
			t.Fatal("history changed existing stance-order bookkeeping")
		}
	})
	for _, kind := range []defs.OrderType{defs.OrderType_UnitAttackSpace, defs.OrderType_UnitAttackZone} {
		t.Run(map[defs.OrderType]string{defs.OrderType_UnitAttackSpace: "space", defs.OrderType_UnitAttackZone: "zone"}[kind], func(t *testing.T) {
			g := rangedGame(t, 1, false)
			p, enemy := g.Human(0), g.Human(1)
			u, target := g.Self(p).Units[0], g.Self(enemy).Units[0]
			area := harness.SpaceKey(1, 0)
			if kind == defs.OrderType_UnitAttackZone {
				area = harness.ZoneKey(1, 0, 0, 0)
			}
			g.Submit(p, harness.Order(kind, []uint64{u.InternalID}, area, false))
			g.EndTurn()
			if g.Self(enemy).Units[0].CombatStats.Health >= target.CombatStats.Health {
				t.Fatal("fixture did not attack")
			}
			actions := g.Self(p).Units[0].TickActions
			if len(actions) < 2 {
				t.Fatalf("tick history has %d actions, want multiple attacks", len(actions))
			}
			attacks := 0
			for _, a := range actions {
				if a.Intent.Kind != "attack" {
					continue
				}
				if a.Execute.Outcome == "blocked" {
					if a.Execute.Reason != "insufficient_stamina" || a.Execute.StaminaSpent != 0 || a.Intent.Target.InternalID != target.InternalID {
						t.Errorf("incorrect rejected attack: %+v", a)
					}
					continue
				}
				attacks++
				if a.Execute.Outcome != "executed" {
					t.Errorf("attack did not execute: %+v", a)
				}
				if a.Tick < 1 || a.Order.OrderType != uint8(kind) || a.Order.TargetID != area || a.Intent.Target != (harness.TickTarget{Kind: "unit", InternalID: target.InternalID}) || a.Execute.Target != a.Intent.Target || a.Execute.StaminaSpent <= 0 {
					t.Errorf("incorrect area attack record: %+v", a)
				}
			}
			if attacks < 2 {
				t.Errorf("recorded %d attacks, want multiple", attacks)
			}
		})
	}
}

func TestRangedAttackStopsAtWeaponRange(t *testing.T) {
	for _, kind := range []defs.OrderType{defs.OrderType_UnitAttackUnit, defs.OrderType_UnitAttackSpace} {
		t.Run(map[defs.OrderType]string{defs.OrderType_UnitAttackUnit: "unit", defs.OrderType_UnitAttackSpace: "space"}[kind], func(t *testing.T) {
			g := rangedGame(t, 2, false)
			owner, enemy := g.Human(0), g.Human(1)
			attacker, target := g.Self(owner).Units[0], g.Self(enemy).Units[0]
			g.Action(owner, "set-unit-behavior", gin.H{"internal_ids": []uint64{attacker.InternalID}, "stance": uint8(defs.UnitStance_PASSIVE), "attack_back": false})
			targetID := int64(target.InternalID)
			if kind == defs.OrderType_UnitAttackSpace {
				targetID = harness.SpaceKey(2, 0)
			}
			g.Submit(owner, harness.Order(kind, []uint64{attacker.InternalID}, targetID, false))
			g.Submit(enemy)
			unit := g.Self(owner).Units[0]
			if unit.Space != (harness.Coord{X: 1}) {
				t.Fatalf("did not stop one space from target: %+v", unit)
			}
			if g.Self(enemy).Units[0].CombatStats.Health >= target.CombatStats.Health {
				t.Fatal("did not attack from range")
			}
			g.EndTurn()
			next := g.Self(owner).Units[0]
			if next.Space != unit.Space || next.Zone != unit.Zone {
				t.Fatalf("kept walking after reaching range: %+v", next)
			}
		})
	}
}
