package combat

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
	"testing"
)

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
