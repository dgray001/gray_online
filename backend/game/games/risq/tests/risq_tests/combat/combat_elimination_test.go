package combat

import (
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"testing"
)

func TestEliminationAndWinCondition(t *testing.T) {
	g := combatGame(t, 13, 1) // 13 is Heavy Infantry (50hp, 9att), 1 is Villager (8hp, 5att)
	p0, p1 := g.Human(0), g.Human(1)
	u0 := g.Self(p0).Units[0]
	u1 := g.Self(p1).Units[0]
	g.Submit(p0, harness.OrderAttackUnit([]uint64{u0.InternalID}, u1.InternalID))
	g.Submit(p1) // Do nothing

	state := g.State(p0)
	if !state.Player(g.PlayerID(p1)).Eliminated {
		t.Errorf("p1 should be eliminated after losing all units and buildings")
	}
	if !g.Base.GameEnded() {
		t.Errorf("game should be ended when only one player remains")
	}
}
