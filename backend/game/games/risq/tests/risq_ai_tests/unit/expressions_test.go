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

func TestExpressionResults(t *testing.T) {
	for expression, want := range map[string]float64{"1 + 2 * 3": 7, "(1 + 2) * 3": 9, "-(-3) + +2": 5, "1 / 4": 0.25, "min(7, max(2, 4))": 4, "var(saved) * 2": 6} {
		model := requireModel(t, fmt.Sprintf(`{"rules":[{"when":{"always":{}},"then":[{"action":"set_var","name":"saved","value":3}]},{"when":{"value_equals":{"value":%q,"amount":%v}},"then":[{"action":"delete_unit"}]}]}`, expression, want))
		view := &fakeView{units: []ai.UnitView{{InternalID: 73}}}
		assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: 12, Subject: 73, Clear: true})
	}
}

func TestExpressionSyntaxRejection(t *testing.T) {
	for _, expression := range []string{"", "1 +", "(1 + 2", "1 + 2)", "min()", "max(1)", "unknown(1)", "missing_bare_name", "1 2", "@"} {
		requireNoop(t, fmt.Sprintf(`{"rules":[{"when":{"always":{}},"then":[{"action":"set_var","name":"result","value":%q}]}]}`, expression))
	}
}

func TestRuntimeDiagnosticFrequency(t *testing.T) {
	var diagnostics bytes.Buffer
	model := jsonModel(t, `{"rules":[{"when":{"always":{}},"then":[{"action":"set_var","name":"failed","value":"1 / (turn - 2)"}]},{"when":{"turn_equals":{"amount":"1 / (turn - 2)"}},"then":[{"action":"delete_unit"}]}]}`, &diagnostics)
	for turn := 0; turn < 2; turn++ {
		decision := model.DecideOrders(&fakeView{turnNumber: 2})
		if len(decision.Orders)+len(decision.Behaviors)+len(decision.BuildingBehaviors) != 0 {
			t.Fatalf("failed condition emitted %+v", decision)
		}
	}
	if strings.Count(diagnostics.String(), "ai expression") != 1 || strings.Count(diagnostics.String(), "ai condition amount") != 2 {
		t.Fatalf("diagnostics %q", diagnostics.String())
	}
}

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
	model := ai.ParseModelWithDiagnostics(raw, rng, io.Discard)
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
	model := ai.ParseModelWithDiagnostics(raw, rng, io.Discard)
	if _, ok := model.(*ai.RulesModel); !ok {
		t.Fatalf("expected RulesModel, got %T", model)
	}
}
