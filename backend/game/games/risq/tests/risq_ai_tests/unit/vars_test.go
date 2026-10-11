package unit

import (
	"bytes"
	"fmt"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"io"
	"math/rand"
	"testing"
)

func TestBuiltinResolverCacheRefreshes(t *testing.T) {
	model := actionsModel(t, `{"action":"move","targets":"coordinate","coordinate":{"x":"var(food)","y":"var(building_count_complete)"}}`)
	for _, c := range []struct {
		food     int
		building bool
		count    int
	}{{20, true, 1}, {7, false, 0}} {
		view := checkedView("Buildings", "IdleUnits", "EligibleUnits")
		view.resources[ai.ResourceFood] = float64(c.food)
		if !c.building {
			view.buildings = nil
		}
		assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: 1, Subject: 91, Place: ai.ZoneRef{Space: ai.Coordinate{X: c.food, Y: c.count}}, Clear: true})
	}
}

func TestCoordinateRounding(t *testing.T) {
	for _, c := range []struct {
		x, y float64
		want ai.Coordinate
	}{{0.49, -0.49, ai.Coordinate{}}, {0.5, -0.5, ai.Coordinate{X: 1, Y: -1}}, {1.5, -1.5, ai.Coordinate{X: 2, Y: -2}}} {
		model := actionsModel(t, fmt.Sprintf(`{"action":"move","targets":"coordinate","coordinate":{"x":%v,"y":%v}}`, c.x, c.y))
		view := checkedView("IdleUnits", "EligibleUnits")
		assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: 1, Subject: 91, Place: ai.ZoneRef{Space: c.want}, Clear: true})
	}
}

func TestBuiltinVariableValues(t *testing.T) {
	for name, value := range map[string]int{"turn": 7, "map_size": 4, "num_players": 3, "enemies_found": 2, "land": 6, "score": 9, "best_enemy_score": 12, "population": 1, "population_limit": 4, "population_max": 10, "food": 20, "wood": 30, "stone": 40, "gold": 50, "resource_food": 20, "unit_cost_food_2": 3, "unit_cost_wood_2": 0, "building_cost_wood_2": 5, "tech_cost_gold_3": 7, "population_military": 1, "population_economic": 0, "idle_units_military": 1, "building_count_complete": 1, "building_count_damaged": 1, "building_count_depleted": 1, "foundation_count_2": 0, "foundation_space_x_2": 0, "bucket_size_missing_name": 0, "tech_researched_4": 1, "resource_remaining_wood": 20, "resource_available_wood": 1} {
		var diagnostics bytes.Buffer
		config := fmt.Sprintf(`{"rules":[{"when":{"always":null},"then":[{"action":"move","targets":"coordinate","coordinate":{"x":"var(%s)","y":0}}]}]}`, name)
		model := jsonModel(t, config, &diagnostics)
		if diagnostics.Len() != 0 {
			t.Fatalf("%s: %s", name, diagnostics.String())
		}
		view := checkedView("Buildings", "IdleUnits", "EligibleUnits", "MaxPopulation", "UnitCost", "BuildCost", "TechCost", "KnownResources", "TechResearched", "NearestResource", "Foundations")
		assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: 1, Subject: 91, Place: ai.ZoneRef{Space: ai.Coordinate{X: value}}, Clear: true})
	}
}

func TestSetVarAction(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	raw := map[string]any{
		"rules": []any{
			map[string]any{
				"when": map[string]any{"always": map[string]any{}},
				"then": []any{
					map[string]any{
						"action": "set_var",
						"name":   "my_var",
						"value":  1.0,
					},
				},
			},
		},
	}
	model := ai.ParseModelWithDiagnostics(raw, rng, io.Discard)
	if _, ok := model.(*ai.RulesModel); !ok {
		t.Fatalf("expected RulesModel, got %T", model)
	}
}

func TestSetVarActionInvalidName(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	raw := map[string]any{
		"rules": []any{
			map[string]any{
				"when": map[string]any{"always": map[string]any{}},
				"then": []any{
					map[string]any{
						"action": "set_var",
						"name":   "123invalid",
						"value":  1.0,
					},
				},
			},
		},
	}
	model := ai.ParseModelWithDiagnostics(raw, rng, io.Discard)
	if _, ok := model.(ai.NoopModel); !ok {
		t.Fatalf("expected NoopModel, got %T", model)
	}
}
