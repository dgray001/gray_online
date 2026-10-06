package combat

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"testing"
)

func TestUnfinishedProducerPreventsDefeat(t *testing.T) {
	g := defeatBehaviorGame(t, 2)
	owner := g.Human(0)
	worker := g.Self(owner).Units[0]
	g.Submit(owner, harness.OrderBuild([]uint64{worker.InternalID}, 22, 0, 0, -1, 0))
	g.EndTurn()
	if !defeatBuilding(g, 0, 22).UnderConstruction {
		t.Fatal("fixture did not leave unfinished barracks")
	}
	g.Submit(owner, harness.Order(defs.OrderType_UnitDelete, []uint64{worker.InternalID}, 0, true))
	g.EndTurn()
	if state := g.Self(owner); state.Eliminated || len(state.Units) != 0 || !defeatBuilding(g, 0, 22).UnderConstruction {
		t.Fatalf("unfinished producer failed: %+v", state)
	}
	g.Submit(owner, harness.Order(defs.OrderType_BuildingDelete, []uint64{defeatBuilding(g, 0, 22).InternalID}, 0, false))
	g.EndTurn()
	if !g.Self(owner).Eliminated || g.Base.GameEnded() {
		t.Fatal("deleted producer prevented defeat or ended three-player game")
	}
}
