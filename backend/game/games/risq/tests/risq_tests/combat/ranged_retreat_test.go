package combat

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
	"testing"
)

func TestRangedDefensiveDoesNotFollowRetreatingTarget(t *testing.T) {
	g := rangedGame(t, 2, false)
	owner, enemy := g.Human(0), g.Human(1)
	attacker, target := g.Self(owner).Units[0], g.Self(enemy).Units[0]
	g.Action(owner, "set-unit-behavior", gin.H{"internal_ids": []uint64{attacker.InternalID}, "stance": uint8(defs.UnitStance_DEFENSIVE), "attack_back": false})
	g.Submit(enemy, harness.OrderMoveZone([]uint64{target.InternalID}, 1, 0, 0, 0))
	g.Submit(owner)
	if g.Self(enemy).Units[0].CombatStats.Health >= target.CombatStats.Health {
		t.Fatal("did not fire when enemy entered range")
	}
	g.Submit(enemy, harness.OrderMoveZone([]uint64{target.InternalID}, 2, 0, 0, 0))
	g.Submit(owner)
	if g.Self(enemy).Units[0].Space != (harness.Coord{X: 2}) {
		t.Fatal("enemy did not retreat beyond range")
	}
	unit := g.Self(owner).Units[0]
	if unit.Space != attacker.Space || unit.Zone != attacker.Zone || len(unit.ActiveOrders) != 0 {
		t.Fatalf("followed retreating target: %+v", unit)
	}
}
