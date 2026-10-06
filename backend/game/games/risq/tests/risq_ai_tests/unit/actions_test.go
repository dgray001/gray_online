package unit

import (
	"math/rand"
	"testing"
	"github.com/dgray001/gray_online/game/games/risq/ai"
)

func parseAndRun(t *testing.T, actionMap map[string]any, view ai.View) ai.Decision {
	rng := rand.New(rand.NewSource(1))
	raw := map[string]any{
		"rules": []any{
			map[string]any{
				"when": map[string]any{"always": map[string]any{}},
				"then": []any{actionMap},
			},
		},
	}
	model := ai.ParseModel(raw, rng)
	if _, ok := model.(*ai.RulesModel); !ok {
		t.Fatalf("expected RulesModel, got %T", model)
	}
	return model.DecideOrders(view)
}

func TestGatherActionOrders(t *testing.T) {
	view := &fakeView{
		units: []ai.UnitView{
			{InternalID: 1, Type: ai.UnitTypeEconomic, Kind: ai.UnitEconomic},
		},
		knownResources: []ai.ResourceView{
			{Category: ai.ResourceWood, AmountLeft: 100},
		},
	}

	action := map[string]any{
		"action": "gather",
		"category": "wood",
	}

	decision := parseAndRun(t, action, view)

	if len(decision.Orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(decision.Orders))
	}
}

func TestCreateActionOrders(t *testing.T) {
	view := &fakeView{
		buildings: []ai.BuildingView{
			{InternalID: 1, Idle: true, Producibles: []ai.Producible{{Kind: ai.ProducibleUnit, ID: 10}}},
		},
		popLimit: 10,
		popCurrent: 0,
		resources: map[ai.ResourceCategory]float64{
			ai.ResourceFood: 100, ai.ResourceWood: 100, ai.ResourceStone: 100, ai.ResourceGold: 100,
		},
	}

	action := map[string]any{
		"action": "create",
		"unit_id": 10.0,
	}

	decision := parseAndRun(t, action, view)

	if len(decision.Orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(decision.Orders))
	}
}

func TestResearchActionOrders(t *testing.T) {
	view := &fakeView{
		buildings: []ai.BuildingView{
			{InternalID: 1, Idle: true, Producibles: []ai.Producible{{Kind: ai.ProducibleTech, ID: 10}}},
		},
		resources: map[ai.ResourceCategory]float64{
			ai.ResourceFood: 100, ai.ResourceWood: 100, ai.ResourceStone: 100, ai.ResourceGold: 100,
		},
	}

	action := map[string]any{
		"action": "research",
		"tech_id": 10.0,
	}

	decision := parseAndRun(t, action, view)

	if len(decision.Orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(decision.Orders))
	}
}

func TestBuildActionOrders(t *testing.T) {
	view := &fakeView{
		units: []ai.UnitView{
			{
				InternalID: 1,
				Type: ai.UnitTypeEconomic,
				Kind: ai.UnitEconomic,
				Builds: []ai.Producible{
					{Kind: ai.ProducibleBuilding, ID: 20},
				},
			},
		},
		resources: map[ai.ResourceCategory]float64{
			ai.ResourceFood: 100, ai.ResourceWood: 100, ai.ResourceStone: 100, ai.ResourceGold: 100,
		},
		buildingAvailable: map[uint32]bool{20: true},
	}

	action := map[string]any{
		"action": "build",
		"building_id": 20.0,
	}

	decision := parseAndRun(t, action, view)

	if len(decision.Orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(decision.Orders))
	}
}

func TestAttackActionOrders(t *testing.T) {
	view := &fakeView{
		units: []ai.UnitView{
			{
				InternalID: 1,
				Type: ai.UnitTypeInfantry,
				Kind: ai.UnitMilitary,
			},
		},
		visibleEnemyUnits: []ai.UnitView{
			{InternalID: 2, Kind: ai.UnitMilitary},
		},
	}

	action := map[string]any{
		"action": "attack",
		"targets": "enemy_units",
	}

	decision := parseAndRun(t, action, view)

	if len(decision.Orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(decision.Orders))
	}
}

func TestGarrisonActionOrders(t *testing.T) {
	view := &fakeView{
		units: []ai.UnitView{
			{InternalID: 1, Kind: ai.UnitEconomic}, // No garrison target logic for tests, just dummy match
		},
		buildings: []ai.BuildingView{
			{InternalID: 2, GarrisonCapacity: 10, GarrisonCount: 0},
		},
	}

	action := map[string]any{
		"action": "garrison",
	}

	decision := parseAndRun(t, action, view)

	if len(decision.Orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(decision.Orders))
	}
}

func TestUngarrisonActionOrders(t *testing.T) {
	garrisonID := uint64(2)
	view := &fakeView{
		units: []ai.UnitView{
			{InternalID: 1, Kind: ai.UnitEconomic, GarrisonedIn: &garrisonID},
		},
	}

	action := map[string]any{
		"action": "ungarrison",
	}

	decision := parseAndRun(t, action, view)

	if len(decision.Orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(decision.Orders))
	}
}
