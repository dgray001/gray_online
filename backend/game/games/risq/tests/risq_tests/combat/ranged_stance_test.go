package combat

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
	"testing"
)

func TestRangedStancesHoldPosition(t *testing.T) {
	for _, c := range []struct {
		name     string
		stance   defs.UnitStance
		distance int
		attacks  bool
	}{
		{"defensive-adjacent", defs.UnitStance_DEFENSIVE, 1, true},
		{"stand-ground-adjacent", defs.UnitStance_STAND_GROUND, 1, true},
		{"passive-adjacent", defs.UnitStance_PASSIVE, 1, false},
		{"defensive-out-of-range", defs.UnitStance_DEFENSIVE, 2, false},
		{"stand-ground-out-of-range", defs.UnitStance_STAND_GROUND, 2, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			g := rangedGame(t, c.distance, false)
			owner, enemy := g.Human(0), g.Human(1)
			attacker, target := g.Self(owner).Units[0], g.Self(enemy).Units[0]
			g.Action(owner, "set-unit-behavior", gin.H{"internal_ids": []uint64{attacker.InternalID}, "stance": uint8(c.stance), "attack_back": false})
			for turn := 0; turn < 2; turn++ {
				g.EndTurn()
				unit := g.Self(owner).Units[0]
				if unit.Space != attacker.Space || unit.Zone != attacker.Zone {
					t.Fatalf("stance moved: %+v", unit)
				}
				if attacked := g.Self(enemy).Units[0].CombatStats.Health < target.CombatStats.Health; attacked != c.attacks {
					t.Fatalf("attacked=%v, want %v", attacked, c.attacks)
				}
				if !c.attacks && len(unit.ActiveOrders) != 0 {
					t.Fatalf("out-of-range auto order: %+v", unit.ActiveOrders)
				}
			}
		})
	}
}
