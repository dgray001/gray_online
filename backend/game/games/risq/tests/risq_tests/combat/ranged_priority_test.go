package combat

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
	"testing"
)

func TestRangedPriorityPrecedesDistance(t *testing.T) {
	for _, stance := range []defs.UnitStance{defs.UnitStance_DEFENSIVE, defs.UnitStance_STAND_GROUND} {
		t.Run(map[defs.UnitStance]string{defs.UnitStance_DEFENSIVE: "defensive", defs.UnitStance_STAND_GROUND: "stand-ground"}[stance], func(t *testing.T) {
			g := rangedGame(t, 1, true)
			owner, enemy := g.Human(0), g.Human(1)
			attacker := g.Self(owner).Units[0]
			g.Action(owner, "set-unit-behavior", gin.H{"internal_ids": []uint64{attacker.InternalID}, "stance": uint8(stance), "attack_back": false, "target_priority": []uint8{uint8(defs.TargetCategory_MILITARY), uint8(defs.TargetCategory_ECONOMIC)}})
			g.EndTurn()
			state := g.Self(enemy)
			if len(state.Units) != 2 {
				t.Fatal("nearby villager killed first")
			}
			for _, unit := range state.Units {
				if unit.UnitID == 13 && unit.CombatStats.Health >= float64(unit.CombatStats.MaxHealth) {
					t.Fatal("adjacent military target ignored")
				}
				if unit.UnitID == 1 && unit.CombatStats.Health != float64(unit.CombatStats.MaxHealth) {
					t.Fatal("nearby villager damaged first")
				}
			}
			unit := g.Self(owner).Units[0]
			if unit.Space != attacker.Space || unit.Zone != attacker.Zone {
				t.Fatalf("ranged unit moved: %+v", unit)
			}
		})
	}
}
