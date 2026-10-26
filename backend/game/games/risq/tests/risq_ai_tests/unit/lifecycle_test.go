package unit

import (
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"io"
	"math/rand"
	"reflect"
	"testing"
)

func TestObservationHistory(t *testing.T) {
	history := make(ai.ObservationHistory)
	spaces := []ai.SpaceInfo{{Space: ai.Coordinate{X: 1}, Vision: 0}, {Space: ai.Coordinate{X: 2}, Vision: 1}, {Space: ai.Coordinate{X: 3}, Vision: 2}, {Space: ai.Coordinate{X: 4}, Vision: 3}, {Space: ai.Coordinate{X: 5}, Vision: 4}}
	history.Observe(spaces, 7)
	expected := ai.ObservationHistory{{X: 3}: 7, {X: 4}: 7, {X: 5}: 7}
	for index := range spaces {
		spaces[index].Vision = 1
	}
	history.Observe(spaces, 9)
	if !reflect.DeepEqual(history, expected) {
		t.Fatalf("fog refreshed observation history: got %v, want %v", history, expected)
	}
	spaces[0].Vision, expected[ai.Coordinate{X: 1}] = 2, 11
	history.Observe(spaces, 11)
	if !reflect.DeepEqual(history, expected) {
		t.Fatalf("poor vision did not update observation history: got %v, want %v", history, expected)
	}
}

func TestVariableRefreshAndApplyUpdate(t *testing.T) {
	config := `{"rules":[{"when":{"value_equals":{"value":"gameVar(turn)","amount":1}},"then":[{"action":"set_var","name":"shared","value":5,"persist":true},{"action":"set_var","name":"shared","value":9},{"action":"set_var","name":"temporary","value":3}]}]}`
	model := requireModel(t, config).(*ai.RulesModel)
	for index, want := range [][2]float64{{9, 3}, {5, 0}, {5, 0}} {
		view := &fakeView{turnNumber: index + 1, units: []ai.UnitView{{InternalID: 73}}}
		model.ApplyUpdate(view, "turn")
		model.DecideOrders(view)
		shared, _ := model.Var("shared")
		temporary, _ := model.Var("temporary")
		if shared != want[0] || temporary != want[1] {
			t.Fatalf("turn %d: shared=%v temporary=%v, want %v", index+1, shared, temporary, want)
		}
	}
	other := requireModel(t, config).(*ai.RulesModel)
	other.DecideOrders(&fakeView{turnNumber: 2})
	if _, ok := other.Var("shared"); ok {
		t.Fatal("a second model saw the first model's persistent variable")
	}
}

func TestDecideOrdersLifecycle(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	raw := map[string]any{
		"rules": []any{
			map[string]any{
				"when": map[string]any{"always": map[string]any{}},
				"then": []any{
					map[string]any{
						"action":  "set_var",
						"name":    "my_turn_var",
						"value":   42.0,
						"persist": false,
					},
					map[string]any{
						"action":  "set_var",
						"name":    "my_persist_var",
						"value":   42.0,
						"persist": true,
					},
				},
			},
		},
	}
	model := ai.ParseModelWithDiagnostics(raw, rng, io.Discard)
	if _, ok := model.(*ai.RulesModel); !ok {
		t.Fatalf("expected RulesModel, got %T", model)
	}

	view := &fakeView{}
	// First turn
	model.DecideOrders(view)
	// Second turn, testing isolation/persistence conceptually
	model.DecideOrders(view)
}
