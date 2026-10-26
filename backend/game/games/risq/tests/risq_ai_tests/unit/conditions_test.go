package unit

import (
	"fmt"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"io"
	"math/rand"
	"testing"
)

const deleteProbe = `{"action":"select","actors":{"kind":"units","rank":"0 - var(actor.uid)","limit":1},"issue":[{"delete":{}}]}`

func TestBooleanConditionShortCircuit(t *testing.T) {
	for _, c := range []struct {
		condition string
		runs      bool
		queries   int
	}{
		{`{"all":[]}`, true, 0}, {`{"any":[]}`, false, 0}, {`{"not":{"not":{"always":null}}}`, true, 0},
		{`{"all":[{"not":{"always":null}},{"tech_researched":{"tech_id":1}}]}`, false, 0},
		{`{"any":[{"always":null},{"tech_researched":{"tech_id":1}}]}`, true, 0},
		{`{"all":[{"always":null},{"tech_researched":{"tech_id":1}}]}`, true, 1},
		{`{"any":[{"not":{"always":null}},{"tech_researched":{"tech_id":1}}]}`, true, 1},
	} {
		model := requireModel(t, fmt.Sprintf(`{"rules":[{"when":%s,"then":[%s]}]}`, c.condition, deleteProbe))
		view := &fakeView{units: []ai.UnitView{{InternalID: 73}}, techResearched: map[uint32]bool{1: true}}
		var calls []orderCall
		if c.runs {
			calls = append(calls, orderCall{Kind: 12, Subject: 73, Clear: true})
		}
		assertCalls(t, model.DecideOrders(view), view, calls...)
		if view.techQueries != c.queries {
			t.Fatalf("%s: queries %d, want %d", c.condition, view.techQueries, c.queries)
		}
	}
}

func TestAlwaysPayloads(t *testing.T) {
	for _, payload := range []string{`{}`, `null`, `false`, `7`, `"ignored"`, `[]`} {
		for _, negated := range []bool{false, true} {
			condition := `{"always":` + payload + `}`
			if negated {
				condition = `{"not":` + condition + `}`
			}
			model := requireModel(t, fmt.Sprintf(`{"rules":[{"when":%s,"then":[%s]}]}`, condition, deleteProbe))
			view := &fakeView{units: []ai.UnitView{{InternalID: 73}}}
			if negated {
				assertCalls(t, model.DecideOrders(view), view)
			} else {
				assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: 12, Subject: 73, Clear: true})
			}
		}
	}
}

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
						"action": "select",
						"actors": map[string]any{"kind": "units"},
						"issue":  []any{map[string]any{"delete": map[string]any{}}}, // produces a dummy order to verify execution
					},
				},
			},
		},
	}
	model := ai.ParseModelWithDiagnostics(raw, rng, io.Discard)
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
