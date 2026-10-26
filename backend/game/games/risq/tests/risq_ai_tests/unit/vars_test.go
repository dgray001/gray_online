package unit

import (
	"bytes"
	"fmt"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"io"
	"math/rand"
	"strings"
	"testing"
)

func moveToX(t *testing.T, expression string, diagnostics io.Writer) (ai.Model, *fakeView) {
	t.Helper()
	config := fmt.Sprintf(`{"rules":[{"when":{"always":null},"then":[{"action":"set_var","name":"probe","value":%q}]}]}`, expression)
	return jsonModel(t, config, diagnostics), checkedView("Buildings", "IdleUnits", "UnitCost", "BuildCost", "TechCost", "MaxPopulation", "TechResearched")
}

func expectX(t *testing.T, model ai.Model, view *fakeView, x int) {
	t.Helper()
	rules := model.(*ai.RulesModel)
	rules.DecideOrders(view)
	if got, _ := rules.Var("probe"); got != float64(x) {
		t.Fatalf("probe = %v, want %d", got, x)
	}
}

func TestGameVarAndResearchedEvaluation(t *testing.T) {
	for expr, val := range map[string]int{
		"gameVar(food)": 20, "gameVar(wood)": 30, "gameVar(stone)": 40, "gameVar(gold)": 50,
		"gameVar(population)": 1, "gameVar(population_limit)": 4, "gameVar(population_max)": 10,
		"gameVar(turn)": 7, "gameVar(map_size)": 4, "gameVar(num_players)": 3,
		"researched(4)": 1, "researched(99)": 0,
	} {
		var diagnostics bytes.Buffer
		model, view := moveToX(t, expr, &diagnostics)
		expectX(t, model, view, val)
	}
}

func TestConfigFunctionReadsConfigValues(t *testing.T) {
	for expression, want := range map[string]int{
		"config(unit, cost_food, 2)":                                  3,
		"config(unit, cost_food, 1 + 1) + config(unit, cost_wood, 2)": 7,
		"config(building, max_health, gameVar(food) - 13)":            90,
		"config(unit, cost_food, 1.6)":                                3,
	} {
		var diagnostics bytes.Buffer
		model, view := moveToX(t, expression, &diagnostics)
		if diagnostics.Len() != 0 {
			t.Fatalf("%s: %s", expression, diagnostics.String())
		}
		expectX(t, model, view, want)
	}
}

func TestConfigFunctionRejectsUnknownNamesAtLoad(t *testing.T) {
	for _, expression := range []string{"config(unit, cost_gems, 2)", "config(castle, max_health, 2)", "config(unit, cost_food)"} {
		model, _ := moveToX(t, expression, io.Discard)
		if _, ok := model.(ai.NoopModel); !ok {
			t.Errorf("%s: expected NoopModel, got %T", expression, model)
		}
	}
}

func TestConfigFunctionUnknownIdEvaluatesToZeroWithoutWarning(t *testing.T) {
	var diagnostics bytes.Buffer
	model, view := moveToX(t, "config(unit, cost_food, 99)", &diagnostics)
	expectX(t, model, view, 0)
	if diagnostics.Len() != 0 {
		t.Errorf("unexpected diagnostics %q", diagnostics.String())
	}
}

func TestConfigFunctionOutOfRangeIdEvaluatesToZeroWithOneWarning(t *testing.T) {
	for _, expression := range []string{"config(unit, cost_food, 0 - 1)", "config(unit, cost_food, 5000000000)"} {
		var diagnostics bytes.Buffer
		model, view := moveToX(t, expression, &diagnostics)
		expectX(t, model, view, 0)
		expectX(t, model, checkedView("Buildings", "IdleUnits"), 0)
		if strings.Count(diagnostics.String(), "ai expression") != 1 {
			t.Errorf("%s: want one warning, got %q", expression, diagnostics.String())
		}
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
