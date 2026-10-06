package sims

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/aisim"
	"testing"
)

func TestGatherAndBuildCarryAcrossTurns(t *testing.T) {
	model := once(`{"action":"gather","category":"food","max":1},{"action":"build","building_id":2,"max":1}`)
	spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":0},"units":[{"id":1,"player":0,"count":2}]},{"x":1,"y":0,"resource":1}]},` + remote
	g := fixture(t, `{"wood":100}`, spaces, "", model, emptyModel)
	g.Run(t, 3)
	p := g.Own(t, 0)
	if p.Resources.Food <= 0 || p.Resources.Wood != 70 {
		t.Fatalf("balance=%+v", p.Resources)
	}
	space := aisim.Decode(t, aisim.Payload(t, g.Player(t, 0))).Space(0, 0)
	for _, row := range space.Zones {
		for _, zone := range row {
			if zone.Resource != nil && zone.Resource.ResourcesLeft >= 300 {
				t.Fatal("food credited without node depletion")
			}
		}
	}
	if len(p.Buildings) != 2 {
		t.Fatalf("buildings=%+v", p.Buildings)
	}
	for _, building := range p.Buildings {
		if building.BuildingID == 2 && building.UnderConstruction {
			t.Fatal("housing never finished")
		}
	}
	assertOrderCount(t, g, 0, defs.OrderType_UnitGather, 1)
	assertOrderCount(t, g, 0, defs.OrderType_UnitBuild, 1)
}

func TestBuildRulesShareOneReservation(t *testing.T) {
	model := once(`{"action":"build","building_id":2,"max":1},{"action":"build","building_id":2,"max":1}`)
	g := fixture(t, `{"wood":30}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":0},"units":[{"id":1,"player":0,"count":2}]}]},`+remote, "", model, emptyModel)
	g.Run(t, 2)
	p := g.Own(t, 0)
	if p.Resources.Wood != 0 || len(p.Buildings) != 2 || p.Buildings[0].UnderConstruction || p.Buildings[1].UnderConstruction {
		t.Fatalf("reservation=%+v buildings=%+v", p.Resources, p.Buildings)
	}
	assertOrderCount(t, g, 0, defs.OrderType_UnitBuild, 1)
}
