package sims

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"testing"
)

func TestGatherRulesRespectFarmCapacity(t *testing.T) {
	spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":3,"player":0},"units":[{"id":1,"player":0,"count":3}]},{"x":1,"y":0,"building":{"id":2,"player":0}}]},` + remote
	g := fixture(t, `{}`, spaces, `,"starting_techs":[1]`, once(`{"action":"gather","category":"food"},{"action":"gather","category":"food"}`), emptyModel)
	g.Run(t, 1)
	assertOrderCount(t, g, 0, defs.OrderType_UnitGather, 2)
	p := g.Own(t, 0)
	if p.Resources.Food != 16 {
		t.Fatalf("farm credits=%+v", p.Resources)
	}
	for _, building := range p.Buildings {
		if building.BuildingID == 3 && building.ResourcesLeft != 184 {
			t.Fatalf("farm=%+v", building)
		}
	}
}
