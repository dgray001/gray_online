package sims

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"testing"
)

func TestCancelledProductionRefundCanBeSpentOnce(t *testing.T) {
	model := `{"rules":[{"when":{"turn_equals":{"amount":1}},"then":[{"action":"create","unit_id":1,"queue":2}]},{"when":{"turn_equals":{"amount":2}},"then":[{"action":"building_stop","orders":["create"]},{"action":"building_stop","orders":["create"]},{"action":"create","unit_id":1,"queue":2}]}]}`
	spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":0},"units":[{"id":1,"player":0,"count":1}]}]},` + remote
	g := fixture(t, `{"food":100}`, spaces, "", model, emptyModel)
	g.Run(t, 2)
	p := g.Own(t, 0)
	if len(p.Units) != 3 || p.Resources.Food != 0 || len(p.Buildings[0].ProductionQueue) != 0 {
		t.Fatalf("refund/spend: units=%+v balance=%+v queue=%+v", p.Units, p.Resources, p.Buildings)
	}
	assertOrderCount(t, g, 0, defs.OrderType_CancelOrder, 1)
	assertOrderCount(t, g, 0, defs.OrderType_BuildingCreate, 3)
}
