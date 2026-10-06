package shape

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestReportRejectedOrderWireShape(t *testing.T) {
	g := configuredGame(t, 3, func() {
		unit := defs.UnitConfigs[1]
		unit.Cost.Food = 2000
		defs.UnitConfigs[1] = unit
	})
	owner := g.Human(1)
	order := harness.OrderProduce([]uint64{centerID(g)}, 1)
	g.Submit(owner, order)
	g.EndTurn()
	report := object(t, player(t, snapshot(g, owner, false), 1)["turn_report"])
	failures := array(t, object(t, report["orders"])["failures"])
	equal(t, len(failures), 1)
	failure := checkShape(t, failures[0], "order_type:n target_id:n reason:s")
	equal(t, failure["order_type"], float64(order.Order_type))
	equal(t, failure["target_id"], float64(order.Target_id))
	equal(t, failure["reason"], "cannot afford unit")
}
