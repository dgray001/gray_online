package shape

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func queueWork(g *harness.Game) (uint64, uint64) {
	g.T.Helper()
	unit, center := unitID(g, 1, 0, 1), centerID(g)
	produce := harness.OrderProduce([]uint64{center}, 1)
	g.Submit(g.Human(1), harness.OrderGather([]uint64{unit}, 0, 0, 1, 0), produce, produce)
	g.EndTurn()
	return unit, center
}

func TestOrderAndProductionQueueWireShapes(t *testing.T) {
	g := shapeGame(t, 4)
	unit, center := queueWork(g)
	owner := player(t, snapshot(g, g.Human(1), false), 1)
	orders := array(t, owner["active_orders"])
	equal(t, len(orders), 2)
	for _, value := range orders {
		order := checkShape(t, value, orderShape)
		equal(t, order["player_id"], float64(1))
		equal(t, len(array(t, order["subjects"])), 1)
	}
	u := entity(t, owner, "units", unit)
	uOrders := array(t, u["active_orders"])
	equal(t, len(uOrders), 1)
	uOrder := checkShape(t, uOrders[0], orderShape)
	equal(t, uOrder["subjects"], []any{float64(unit)})
	equal(t, uOrder["target_id"], float64(harness.ZoneKey(0, 0, 1, 0)))
	equal(t, uOrder["clear_previous_orders"], false)
	b := entity(t, owner, "buildings", center)
	queue := array(t, b["production_queue"])
	equal(t, len(queue), 1)
	item := checkShape(t, queue[0], "kind:n item_id:n stamina_remaining:n order_internal_id:n")
	bOrders := array(t, b["active_orders"])
	equal(t, len(bOrders), 1)
	order := checkShape(t, bOrders[0], orderShape)
	equal(t, order["subjects"], []any{float64(center)})
	equal(t, order["target_id"], float64(1))
	equal(t, item["order_internal_id"], order["internal_id"])
	equal(t, item["kind"], float64(1))
	equal(t, item["item_id"], float64(1))
}
