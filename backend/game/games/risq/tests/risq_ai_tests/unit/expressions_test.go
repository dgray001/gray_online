package unit

import (
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"math/rand"
	"testing"
)

func TestParseAmount(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	raw := map[string]any{
		"rules": []any{
			map[string]any{
				"when": map[string]any{
					"resource_equals": map[string]any{
						"category": "wood",
						"amount":   "1 + 2 * 3",
					},
				},
				"then": []any{},
			},
		},
	}
	model := ai.ParseModel(raw, rng)
	if _, ok := model.(*ai.RulesModel); !ok {
		t.Fatalf("expected RulesModel, got %T", model)
	}
}

func TestParseExpressionVariables(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	raw := map[string]any{
		"rules": []any{
			map[string]any{
				"when": map[string]any{
					"resource_equals": map[string]any{
						"category": "wood",
						"amount":   "turn + population + var(my_var)",
					},
				},
				"then": []any{},
			},
		},
	}
	model := ai.ParseModel(raw, rng)
	if _, ok := model.(*ai.RulesModel); !ok {
		t.Fatalf("expected RulesModel, got %T", model)
	}
}
