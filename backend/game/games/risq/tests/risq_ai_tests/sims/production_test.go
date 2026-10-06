package sims

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"testing"
)

func TestHireAndProductionChargeOnce(t *testing.T) {
	model := once(`{"action":"hire","unit_id":11,"max":1},{"action":"create","unit_id":1,"queue":1}`)
	spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":0},"units":[{"id":1,"player":0,"count":1}]}]},` + remote
	g := fixture(t, `{"food":100,"gold":500}`, spaces, `,"starting_techs":[4]`, model, emptyModel)
	g.Run(t, 2)
	p := g.Own(t, 0)
	if len(p.Units) != 3 || p.Resources.Food != 50 || p.Resources.Gold != 409 {
		t.Fatalf("units=%+v balance=%+v", p.Units, p.Resources)
	}
	if len(p.Buildings[0].ProductionQueue) != 0 {
		t.Fatalf("unfinished production=%+v", p.Buildings)
	}
	assertOrderCount(t, g, 0, defs.OrderType_BuyMercenary, 1)
	assertOrderCount(t, g, 0, defs.OrderType_BuildingCreate, 1)
}
