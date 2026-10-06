package orders

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestBuildingCapabilitiesConstrainOrders(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	center := buildingID(g, p, 1)
	g.SubmitRejected(p, harness.OrderProduce([]uint64{center}, 11), "Building id 1 cannot produce unit id 11")
	g.SubmitRejected(p, harness.Order(defs.OrderType_BuildingResearch, []uint64{center}, 2, false), "Building id 1 cannot research tech id 2")
	g.SubmitRejected(p, harness.Order(defs.OrderType_UnitRenew, unitIDs(g, p, 1)[:1], int64(center), false), "Building id 1 is not gatherable")
	unarmed := buildingID(g, p, 11)
	g.SubmitRejected(p, harness.Order(defs.OrderType_BuildingAttackUnit, []uint64{unarmed}, int64(unitIDs(g, g.Human(1), 1)[0]), false), fmt.Sprintf("Building id %d cannot attack", unarmed))
}
