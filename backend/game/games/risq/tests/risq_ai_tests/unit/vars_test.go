package unit

import (
	"math/rand"
	"testing"
	"github.com/dgray001/gray_online/game/games/risq/ai"
)

func TestSetVarAction(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	raw := map[string]any{
		"rules": []any{
			map[string]any{
				"when": map[string]any{"always": map[string]any{}},
				"then": []any{
					map[string]any{
						"action": "set_var",
						"name": "my_var",
						"value": 1.0,
					},
				},
			},
		},
	}
	model := ai.ParseModel(raw, rng)
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
						"name": "123invalid",
						"value": 1.0,
					},
				},
			},
		},
	}
	model := ai.ParseModel(raw, rng)
	if _, ok := model.(ai.NoopModel); !ok {
		t.Fatalf("expected NoopModel, got %T", model)
	}
}
