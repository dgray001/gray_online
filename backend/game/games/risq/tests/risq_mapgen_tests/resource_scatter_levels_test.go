package risq_mapgen_tests

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
)

type scatterTestParams struct {
	Chance          json.RawMessage    `json:"chance"`
	CategoryWeights map[string]float64 `json:"category_weights"`
}

func scatterChance(t *testing.T, raw json.RawMessage, multiplier float64) float64 {
	t.Helper()
	var chance float64
	if json.Unmarshal(raw, &chance) == nil {
		return chance
	}
	var expression string
	if err := json.Unmarshal(raw, &expression); err != nil {
		t.Fatal(err)
	}
	chance, err := util.EvalExpr(expression, map[string]float64{"outside_resource_multiplier": multiplier})
	if err != nil {
		t.Fatal(err)
	}
	return chance
}

func mapScatters(t *testing.T, name string) []scatterTestParams {
	t.Helper()
	data, err := defs.ReadConfigFile("maps", "scripted", name+".json")
	if err != nil {
		t.Fatal(err)
	}
	var steps []struct {
		Step   string
		Params scatterTestParams
	}
	if err := json.Unmarshal(data, &steps); err != nil {
		t.Fatal(err)
	}
	scatters := make([]scatterTestParams, 0)
	for _, step := range steps {
		if step.Step == "resource_scatter" {
			scatters = append(scatters, step.Params)
		}
	}
	return scatters
}

func TestOutsideFoodScalingKeepsWoodFixed(t *testing.T) {
	useTestScriptMaps(t)
	for _, name := range []string{"hexagon", "ring", "rectangle", "triangle"} {
		scatters := mapScatters(t, name)
		if len(scatters) != 2 || scatters[0].CategoryWeights["food"] != 0 || scatters[1].CategoryWeights["food"] == 0 {
			t.Fatalf("%s does not separate wood and food", name)
		}
		for multiplier, want := range map[float64]float64{0.5: 0.0125, 1: 0.025, 2: 0.05, 4: 0.1} {
			wood := scatterChance(t, scatters[0].Chance, multiplier)
			food := (1 - wood) * scatterChance(t, scatters[1].Chance, multiplier)
			if wood != 0.125 || math.Abs(food-want) > 1e-12 {
				t.Errorf("%s multiplier %v: wood %v food %v, want 0.125 and %v", name, multiplier, wood, food, want)
			}
		}
	}
}
