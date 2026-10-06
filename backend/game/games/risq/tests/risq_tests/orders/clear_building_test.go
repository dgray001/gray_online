package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func TestClearPreviousPreservesProductionAndResearch(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	b := buildingID(g, p, 1)
	g.Submit(p, harness.Order(defs.OrderType_BuildingResearch, []uint64{b}, 1, false), harness.OrderProduce([]uint64{b}, 1), harness.Order(defs.OrderType_BuildingDelete, []uint64{b}, 0, false))
	g.EndTurn()
	if len(g.Self(p).ActiveOrders) != 3 {
		t.Fatal("setup did not leave research, production and deletion queued")
	}
	order := harness.OrderProduce([]uint64{b}, 1)
	order.Clear_previous_orders = true
	g.Submit(p, order)
	g.EndTurn()
	if !g.Self(p).ResearchedTechs[1] || g.Self(p).Resources.Food != 850 || g.Self(p).Resources.Wood != 950 {
		t.Fatal("clear cancelled or refunded immune work")
	}
	for _, active := range g.Self(p).ActiveOrders {
		if active.OrderType == uint8(defs.OrderType_BuildingDelete) {
			t.Fatal("clear preserved cancellable building work")
		}
	}
	if queue := entity(g, p, "buildings", b)["production_queue"].([]gin.H); len(queue) != 2 {
		t.Fatalf("production queue lost items: %v", queue)
	}
	for range 3 {
		g.EndTurn()
	}
	if len(g.Self(p).Units) != 5 || len(g.Self(p).ActiveOrders) != 0 {
		t.Fatal("preserved production did not complete")
	}
}
