package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestHousingSupportsPopulationOnlyWhileCompletedAndAlive(t *testing.T) {
	g := buildGame(t, 1)
	p := g.Human(0)
	g.Submit(p, housingOrders(g, p, 0)...)
	g.EndTurn()
	if g.Self(p).PopulationLimit != 0 || !centerHousing(g, p).UnderConstruction {
		t.Fatal("unfinished housing provided population support")
	}
	g.EndTurn()
	if g.Self(p).PopulationLimit != 5 || centerHousing(g, p).UnderConstruction {
		t.Fatal("completed housing did not provide population support")
	}
	g.Submit(p, harness.Order(defs.OrderType_BuildingDelete, []uint64{centerHousing(g, p).InternalID}, 0, false))
	g.EndTurn()
	if g.Self(p).PopulationLimit != 0 || len(g.Self(p).Buildings) != 0 {
		t.Error("deleted housing retained population support")
	}
}
