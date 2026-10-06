package sims

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"testing"
)

func TestCancelledResearchRefundCanFundProduction(t *testing.T) {
	model := `{"rules":[{"when":{"turn_equals":{"amount":1}},"then":[{"action":"research","tech_id":1}]},{"when":{"turn_equals":{"amount":2}},"then":[{"action":"building_stop","orders":["research"]},{"action":"building_stop","orders":["research"]},{"action":"create","unit_id":1}]}]}`
	spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":0},"units":[{"id":1,"player":0,"count":1}]}]},` + remote
	g := fixture(t, `{"food":50,"wood":50}`, spaces, "", model, emptyModel)
	g.Run(t, 2)
	p := g.Own(t, 0)
	if len(p.Units) != 2 || p.Resources.Food != 0 || p.Resources.Wood != 50 || p.ResearchedTechs[1] {
		t.Fatalf("research refund: player=%+v balance=%+v", p, p.Resources)
	}
	assertOrderCount(t, g, 0, defs.OrderType_CancelOrder, 1)
	assertOrderCount(t, g, 0, defs.OrderType_BuildingCreate, 1)
}
