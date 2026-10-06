package combat

import (
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"testing"
)

func TestKillCredit(t *testing.T) {
	g := combatGame(t, 13, 11) // 13 (50hp, 18att) vs 11 (11hp)
	p0, p1 := g.Human(0), g.Human(1)
	u0 := g.Self(p0).Units[0]
	u1 := g.Self(p1).Units[0]
	g.Submit(p0, harness.OrderAttackUnit([]uint64{u0.InternalID}, u1.InternalID))
	g.Submit(p1) // do nothing

	r0 := g.Risq.Results().Players[g.PlayerID(p0)]
	r1 := g.Risq.Results().Players[g.PlayerID(p1)]
	if r0.Kills != 1 {
		t.Errorf("p0 should have 1 kill, got %v", r0.Kills)
	}
	if r1.UnitsLost != 1 {
		t.Errorf("p1 should have 1 unit lost, got %v", r1.UnitsLost)
	}
}
