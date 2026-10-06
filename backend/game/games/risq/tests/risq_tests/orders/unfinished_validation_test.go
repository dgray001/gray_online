package orders

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestUnfinishedBuildingCannotProduceResearchOrBeRepaired(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	workers := unitIDs(g, p, 1)[:1]
	g.Submit(p, harness.OrderBuild(workers, 2, 0, 0, -1, 1))
	g.EndTurn()
	b := buildingID(g, p, 2)
	if entity(g, p, "buildings", b)["under_construction"] != true {
		t.Fatal("setup building finished too soon")
	}
	for _, typ := range []defs.OrderType{defs.OrderType_BuildingCreate, defs.OrderType_BuildingResearch} {
		g.SubmitRejected(p, harness.Order(typ, []uint64{b}, 1, false), fmt.Sprintf("Building id %d is still under construction", b))
	}
	g.SubmitRejected(p, harness.Order(defs.OrderType_UnitRepair, workers, int64(b), false), "Cannot repair a building under construction")
}
