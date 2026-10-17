package sims

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/aisim"
	"github.com/gin-gonic/gin"
)

func assertBehaviorSequence(t *testing.T, g *aisim.Game, slot int) {
	t.Helper()
	var kinds []string
	for _, action := range g.Trace {
		if action.Ai_id == int(g.Player(t, slot).GetAiId()) {
			kinds = append(kinds, action.Kind)
		}
	}
	want := []string{"set-unit-behavior", "set-building-behavior", "submit-orders"}
	if !reflect.DeepEqual(kinds, want) {
		t.Fatalf("sequence=%v, want %v", kinds, want)
	}
	payload := aisim.Payload(t, g.Player(t, slot))
	assertBehaviorPayload(t, payload, slot)
}

func assertBehaviorPayload(t *testing.T, payload gin.H, slot int) {
	t.Helper()
	var state struct {
		Players []struct {
			Player struct {
				ID int `json:"player_id"`
			} `json:"player"`
			Units []struct {
				Stance     uint8 `json:"stance"`
				AttackBack bool  `json:"attack_back"`
			} `json:"units"`
			Buildings []struct {
				AutoAttack bool `json:"auto_attack"`
			} `json:"buildings"`
		} `json:"players"`
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}
	for _, player := range state.Players {
		if player.Player.ID != slot {
			continue
		}
		if len(player.Units) != 1 || player.Units[0].Stance != 1 || player.Units[0].AttackBack {
			t.Fatalf("unit behavior=%+v", player.Units)
		}
		if len(player.Buildings) != 1 || player.Buildings[0].AutoAttack {
			t.Fatalf("building behavior=%+v", player.Buildings)
		}
		return
	}
	t.Fatalf("missing player %d", slot)
}

func TestExplicitAttackAndBehaviorReachEngine(t *testing.T) {
	model := once(`{"action":"set_unit_behavior","stance":"passive","attack_back":false},{"action":"set_building_behavior","auto_attack":false},{"action":"attack","target":"economic","max":1}`)
	spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":0},"units":[{"id":101,"player":0,"count":1},{"id":1,"player":1,"count":1}]}]},` + remote
	g := fixture(t, `{}`, spaces, "", model, emptyModel)
	g.Run(t, 1)
	assertOrderCount(t, g, 0, defs.OrderType_UnitAttackUnit, 1)
	if len(g.Own(t, 1).Units) != 1 {
		t.Fatalf("enemy survived: %+v", g.Own(t, 1).Units)
	}
	if g.Risq.Results().Players[0].Kills != 1 {
		t.Fatalf("results=%+v", g.Risq.Results())
	}
	assertBehaviorSequence(t, g, 0)
}

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

func TestHireAndProductionSharePopulationReservation(t *testing.T) {
	for _, hireFirst := range []bool{true, false} {
		t.Run(map[bool]string{true: "hire-first", false: "create-first"}[hireFirst], func(t *testing.T) {
			hire, create := `{"action":"hire","unit_id":101,"max":2}`, `{"action":"create","unit_id":1,"queue":2}`
			actions := create + "," + hire
			if hireFirst {
				actions = hire + "," + create
			}
			spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":0},"units":[{"id":1,"player":0,"count":4}]}]},` + remote
			g := fixture(t, `{"food":100,"gold":500}`, spaces, `,"starting_techs":[4]`, once(actions), emptyModel)
			g.Run(t, 2)
			p := g.Own(t, 0)
			if len(p.Units) != 5 {
				t.Fatalf("population=%+v", p.Units)
			}
			creates, hires, food, gold := 1, 0, 50.0, 500.0
			if hireFirst {
				creates, hires, food, gold = 0, 1, 100, 409
			}
			assertOrderCount(t, g, 0, defs.OrderType_BuildingCreate, creates)
			assertOrderCount(t, g, 0, defs.OrderType_BuyMercenary, hires)
			if p.Resources.Food != food || p.Resources.Gold != gold {
				t.Fatalf("balance=%+v", p.Resources)
			}
		})
	}
}

func TestHireAndProductionChargeOnce(t *testing.T) {
	model := once(`{"action":"hire","unit_id":101,"max":1},{"action":"create","unit_id":1,"queue":1}`)
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
