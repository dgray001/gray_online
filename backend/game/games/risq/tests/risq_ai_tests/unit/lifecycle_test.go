package unit

import (
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"math/rand"
	"testing"
)

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
	model := ai.ParseModel(raw, rng)
	if _, ok := model.(*ai.RulesModel); !ok {
		t.Fatalf("expected RulesModel, got %T", model)
	}

	view := &fakeView{}
	// First turn
	model.DecideOrders(view)
	// Second turn, testing isolation/persistence conceptually
	model.DecideOrders(view)
}
