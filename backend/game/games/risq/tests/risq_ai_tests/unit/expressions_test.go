package unit

import (
	"bytes"
	"fmt"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/util"
	"io"
	"math/rand"
	"strings"
	"testing"
)

func TestSetVarPreservesStateOnFailureAndAcceptsZero(t *testing.T) {
	for _, persist := range []bool{false, true} {
		config := rule(fmt.Sprintf(`{"action":"set_var","name":"saved","value":7,"persist":%t},
			{"action":"set_var","name":"saved","value":"1 / (turn - turn)","persist":%t},
			{"action":"set_var","name":"zero","value":7,"persist":%t},
			{"action":"set_var","name":"zero","value":"turn - turn","persist":%t}`, persist, persist, persist, persist))
		got := decide(t, config, loopFixture(), "saved", "zero")
		if got["saved"] != 7 || got["zero"] != 0 {
			t.Errorf("persist %v: values %v, want saved=7 and zero=0", persist, got)
		}
	}
}

func TestExpressionComparisons(t *testing.T) {
	for expression, want := range map[string]float64{
		"2==2": 1, "2==3": 0, "2!=3": 1, "2!=2": 0,
		"2<3": 1, "2<2": 0, "3>2": 1, "2>2": 0,
		"2<=2": 1, "3<=2": 0, "2>=2": 1, "2>=3": 0,
		"1 + 2 * 3 >= 7": 1, "2 == 1 + 1": 1, "-2 < -1": 1,
		"2 * (3 < 4) + 1": 3, "min(2 == 2, 3 < 4)": 1, "max(2 > 3, 5 >= 5)": 1,
	} {
		got, err := util.EvalExpr(expression, nil)
		if err != nil || got != want {
			t.Errorf("%s = %v, %v; want %v", expression, got, err, want)
		}
	}
}

func TestExpressionComparisonErrors(t *testing.T) {
	for _, expression := range []string{"1 = 1", "1 ! 1", "1 === 1", "1 <> 2", "1 <", "<= 1", "1 < 2 < 3"} {
		if _, err := util.EvalExpr(expression, nil); err == nil {
			t.Errorf("invalid comparison accepted: %s", expression)
		}
		requireNoop(t, rule(fmt.Sprintf(`{"action":"set_var","name":"x","value":%q}`, expression)))
	}
}

func TestComparisonsFilterSelectedActors(t *testing.T) {
	view := selectTestView{loopTestView: loopFixture()}
	view.spaces = lineOfSpaces(0)
	config := rule(`{"action":"set_var","name":"threshold","value":6},
		{"action":"select","pick":{"kind":"spaces"},"actors":{"kind":"units","where":{"if":["var(actor.health) >= var(threshold)","var(actor.health) < 9"]}},"issue":[{"move":{}}]}`)
	if orders := selectOrders(t, config, view); len(orders) != 1 || orders[0].Subjects[0] != 2 {
		t.Fatalf("orders = %+v, want only the actor with health 6", orders)
	}
}

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

func TestValidationContinuesAfterZeroDivisors(t *testing.T) {
	var visited []string
	expression := "1 / (var(first) - var(first)) + random() + randomInt(1 / 0, 2 / 0) + min(var(after), round(1 / 0))"
	if err := util.ValidateExpr(expression, nil, func(name string) { visited = append(visited, name) }); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(visited, ","); got != "first,first,after" {
		t.Fatalf("visited %q, want all references including those after zero divisors", got)
	}
	if _, err := util.EvalExpr("1 / 0", nil); err == nil || err.Error() != "division by zero" {
		t.Fatalf("runtime division error = %v", err)
	}
}

func TestInvalidExpressionsAfterZeroDivisorsAreRejected(t *testing.T) {
	for _, expression := range []string{
		"1 / 0 +", "1 / 0 + unknown(1)", "1 / 0 + missing_bare_name",
		"1 / 0 + round()", "1 / 0 + min(1)", "1 / 0 + random(1)", "1 / 0 + randomInt(1)",
		"1 / 0 + var(1)", "1 / 0 + var(target.uid)",
	} {
		requireNoop(t, rule(fmt.Sprintf(`{"action":"set_var","name":"x","value":%q}`, expression)))
	}
	for _, selector := range []string{
		`"limit":"1 / 0 + var(target.uid)"`,
		`"rank":"1 / 0 + var(target.no_such_field)"`,
	} {
		requireNoop(t, rule(`{"action":"select","pick":{"kind":"units",`+selector+`}}`))
	}
}

func TestUnknownVariablesAfterZeroDivisorsAreReported(t *testing.T) {
	var diagnostics bytes.Buffer
	model := jsonModel(t, rule(`{"action":"set_var","name":"x","value":"1 / 0 + var(misspelled)"}`), &diagnostics)
	if _, ok := model.(*ai.RulesModel); !ok {
		t.Fatalf("expected valid syntax, got %T", model)
	}
	if !strings.Contains(diagnostics.String(), "var(misspelled)") {
		t.Fatalf("missing variable diagnostic: %q", diagnostics.String())
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

func TestFailedIfConditionsSkipBothBranches(t *testing.T) {
	for _, when := range []string{
		`{"value_equals":{"value":"1 / 0","amount":0}}`,
		`{"not":{"value_equals":{"value":"1 / 0","amount":0}}}`,
	} {
		config := rule(`{"action":"set_var","name":"branch","value":0},
			{"action":"if","when":` + when + `,"then":[{"action":"set_var","name":"branch","value":1}],
			"else":[{"action":"set_var","name":"branch","value":2}]}`)
		if got := decide(t, config, loopFixture(), "branch")["branch"]; got != 0 {
			t.Errorf("when %s: branch = %v, want neither branch", when, got)
		}
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
