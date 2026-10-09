package unit

import (
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"math/rand"
	"testing"
)

func runCondition(t *testing.T, condition map[string]any, view ai.View) []ai.Order {
	rng := rand.New(rand.NewSource(1))
	raw := map[string]any{
		"rules": []any{
			map[string]any{
				"when": condition,
				"then": []any{
					map[string]any{
						"action": "set_var",
						"name":   "triggered",
						"value":  1.0,
					},
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

func TestConditionAlways(t *testing.T) {
	view := &fakeView{
		units: []ai.UnitView{{InternalID: 1}},
	}
	orders := runCondition(t, map[string]any{"always": map[string]any{}}, view)
	if len(orders) == 0 {
		t.Fatal("expected 'always' condition to trigger action")
	}
}

func TestConditionNot(t *testing.T) {
	view := &fakeView{
		units: []ai.UnitView{{InternalID: 1}},
	}
	orders := runCondition(t, map[string]any{"not": map[string]any{"always": map[string]any{}}}, view)
	if len(orders) != 0 {
		t.Fatal("expected 'not(always)' condition to suppress action")
	}
}

func TestConditionTechResearched(t *testing.T) {
	viewTrue := &fakeView{
		units:          []ai.UnitView{{InternalID: 1}},
		techResearched: map[uint32]bool{42: true},
	}
	viewFalse := &fakeView{
		units:          []ai.UnitView{{InternalID: 1}},
		techResearched: map[uint32]bool{42: false},
	}

	cond := map[string]any{"tech_researched": map[string]any{"tech_id": 42.0}}

	orders := runCondition(t, cond, viewTrue)
	if len(orders) == 0 {
		t.Fatal("expected condition to be true")
	}

	orders = runCondition(t, cond, viewFalse)
	if len(orders) != 0 {
		t.Fatal("expected condition to be false")
	}
}

func TestConditionResourceAvailable(t *testing.T) {
	viewTrue := &fakeView{
		units:          []ai.UnitView{{InternalID: 1}},
		knownResources: []ai.ResourceView{{Category: ai.ResourceWood}},
		buildings:      []ai.BuildingView{{BuildingID: 1, Location: ai.ZoneRef{}}}, // Needs home
	}
	viewFalse := &fakeView{
		units:          []ai.UnitView{{InternalID: 1}},
		knownResources: []ai.ResourceView{{Category: ai.ResourceGold}}, // Different category
		buildings:      []ai.BuildingView{{BuildingID: 1, Location: ai.ZoneRef{}}},
	}

	cond := map[string]any{"resource_available": map[string]any{"category": "wood"}}

	orders := runCondition(t, cond, viewTrue)
	if len(orders) == 0 {
		t.Fatal("expected condition to be true")
	}

	orders = runCondition(t, cond, viewFalse)
	if len(orders) != 0 {
		t.Fatal("expected condition to be false")
	}
}
