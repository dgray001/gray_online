package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func refundOrders(t *testing.T, kind string) (*harness.Game, int, defs.OrderFromFrontend, defs.OrderFromFrontend) {
	bank := `{"food":70,"wood":40}`
	if kind == "research" {
		bank = `{"stone":100,"gold":100}`
	}
	if kind == "foundation" {
		bank = `{"wood":30}`
	}
	g := orderGame(t, bank, "")
	p := g.Human(0)
	first := harness.OrderProduce([]uint64{buildingID(g, p, 22)}, 12)
	if kind == "research" {
		first = harness.Order(defs.OrderType_BuildingResearch, []uint64{buildingID(g, p, 11)}, 2, false)
	}
	if kind == "foundation" {
		first = harness.OrderBuild(unitIDs(g, p, 1)[:1], 2, 3, 0, 0, 0)
	}
	g.Submit(p, first)
	g.EndTurn()
	state := g.Self(p)
	if len(state.ActiveOrders) != 1 {
		t.Fatalf("refund setup has no active order: %+v", state)
	}
	refund := harness.Order(defs.OrderType_CancelOrder, nil, int64(state.ActiveOrders[0].InternalID), false)
	spend := first
	if kind == "research" {
		spend.Target_id = 3
	}
	if kind == "foundation" {
		if len(state.PlannedFoundations) != 1 {
			t.Fatal("refund setup has no planned foundation")
		}
		refund = harness.Order(defs.OrderType_CancelFoundation, nil, harness.ZoneKey(3, 0, 0, 0), false)
		spend = harness.OrderBuild(first.Subjects, 2, 0, 0, -1, 1)
	}
	return g, p, refund, spend
}
