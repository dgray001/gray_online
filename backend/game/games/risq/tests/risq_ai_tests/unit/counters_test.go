package unit

import (
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"math/rand"
	"testing"
)

func runCounterCondition(t *testing.T, condition map[string]any, view ai.View) []ai.Order {
	rng := rand.New(rand.NewSource(1))
	raw := map[string]any{
		"rules": []any{
			map[string]any{
				"when": condition,
				"then": []any{
					map[string]any{
						"action": "delete_unit", // produces a dummy order to verify execution
					},
				},
			},
		},
	}
	model := ai.ParseModel(raw, rng)
	if _, ok := model.(*ai.RulesModel); !ok {
		t.Fatalf("expected RulesModel, got %T", model)
	}
	return model.DecideOrders(view).Orders
}

func TestCounterResource(t *testing.T) {
	view := &fakeView{
		units: []ai.UnitView{{InternalID: 1}},
		resources: map[ai.ResourceCategory]float64{
			ai.ResourceWood: 150.0,
		},
	}

	condEqual := map[string]any{"resource_equals": map[string]any{"category": "wood", "amount": 150.0}}
	condMore := map[string]any{"resource_at_least": map[string]any{"category": "wood", "amount": 100.0}}
	condLess := map[string]any{"resource_at_most": map[string]any{"category": "wood", "amount": 50.0}}

	if len(runCounterCondition(t, condEqual, view)) == 0 {
		t.Fatal("expected resource_equals 150.0 to be true")
	}
	if len(runCounterCondition(t, condMore, view)) == 0 {
		t.Fatal("expected resource_at_least 100.0 to be true")
	}
	if len(runCounterCondition(t, condLess, view)) != 0 {
		t.Fatal("expected resource_at_most 50.0 to be false")
	}
}

func TestCounterPopulation(t *testing.T) {
	view := &fakeView{
		units: []ai.UnitView{
			{InternalID: 1, Type: ai.UnitTypeEconomic},
			{InternalID: 2, Type: ai.UnitTypeInfantry},
		},
	}

	condEqual := map[string]any{"population_equals": map[string]any{"amount": 2.0}}
	condLess := map[string]any{"population_at_most": map[string]any{"amount": 1.0}}

	if len(runCounterCondition(t, condEqual, view)) == 0 {
		t.Fatal("expected population_equals 2.0 to be true")
	}
	if len(runCounterCondition(t, condLess, view)) != 0 {
		t.Fatal("expected population_at_most 1.0 to be false")
	}
}

func TestCounterIdleUnits(t *testing.T) {
	view := &fakeView{
		units: []ai.UnitView{
			{InternalID: 1, Type: ai.UnitTypeEconomic},
			{InternalID: 2, Type: ai.UnitTypeEconomic},
		},
	}
	// fakeView's IdleUnits returns all units currently

	condEqual := map[string]any{"idle_units_equals": map[string]any{"amount": 2.0}}
	condLess := map[string]any{"idle_units_at_most": map[string]any{"amount": 1.0}}

	if len(runCounterCondition(t, condEqual, view)) == 0 {
		t.Fatal("expected idle_units_equals 2.0 to be true")
	}
	if len(runCounterCondition(t, condLess, view)) != 0 {
		t.Fatal("expected idle_units_at_most 1.0 to be false")
	}
}
