package unit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"io"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

func TestConcurrentDiagnosticOwnership(t *testing.T) {
	for index := 0; index < 12; index++ {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			t.Parallel()
			name := fmt.Sprintf("unknown_%d", index)
			var diagnostics bytes.Buffer
			config := fmt.Sprintf(`{"rules":[{"when":{"always":null},"then":[{"action":"set_var","name":"result","value":"var(%s)"}]}]}`, name)
			model := jsonModel(t, config, &diagnostics)
			if _, ok := model.(*ai.RulesModel); !ok || strings.Count(diagnostics.String(), "\n") != 1 || !strings.Contains(diagnostics.String(), "var("+name+")") {
				t.Fatalf("model %T, diagnostics %q", model, diagnostics.String())
			}
			for turn := 0; turn < 2; turn++ {
				if decision := model.DecideOrders(&fakeView{}); len(decision.Orders) != 0 || strings.Count(diagnostics.String(), "\n") != 1 {
					t.Errorf("decision contaminated another model: %+v/%q", decision, diagnostics.String())
				}
			}
		})
	}
}

func jsonModel(t *testing.T, config string, diagnostics io.Writer) ai.Model {
	t.Helper()
	var raw map[string]any
	if err := json.Unmarshal([]byte(config), &raw); err != nil {
		t.Fatal(err)
	}
	return ai.ParseModelWithDiagnostics(raw, rand.New(rand.NewSource(1)), diagnostics)
}

func requireModel(t *testing.T, config string) ai.Model {
	t.Helper()
	model := jsonModel(t, config, io.Discard)
	if _, ok := model.(*ai.RulesModel); !ok {
		t.Fatalf("expected RulesModel, got %T: %s", model, config)
	}
	return model
}

func TestBucketDepthAndRestoration(t *testing.T) {
	for _, next := range []string{"a", "b"} {
		var diagnostics bytes.Buffer
		config := `{"rules":[{"when":{"always":null},"then":[{"action":"set_bucket","bucket":"a","size":1,"task":{"action":"run_bucket","bucket":"` + next + `"}},{"action":"fill_bucket","bucket":"a"},{"action":"set_bucket","bucket":"b","size":1,"task":{"action":"run_bucket","bucket":"a"}},{"action":"fill_bucket","bucket":"b"},{"action":"run_bucket","bucket":"a"},{"action":"move","targets":"coordinate","coordinate":{"x":7,"y":0},"max":1}]}]}`
		model := jsonModel(t, config, &diagnostics)
		for turn := 0; turn < 2; turn++ {
			view := &fakeView{units: []ai.UnitView{{InternalID: 73}, {InternalID: 91}}}
			assertScenarioCalls(t, "cycle "+next, model.DecideOrders(view), view, []namedCall{{"independent action after cycle", orderCall{Kind: 1, Subject: 73, Place: ai.ZoneRef{Space: ai.Coordinate{X: 7}}, Clear: true}}})
			if strings.Count(diagnostics.String(), "max_bucket_depth") != 1 {
				t.Errorf("cycle warning count: %q", diagnostics.String())
			}
		}
	}
	rules := `"rules":[{"when":{"always":{}},"then":[{"action":"set_bucket","bucket":"a","size":1,"task":{"action":"run_bucket","bucket":"b"}},{"action":"fill_bucket","bucket":"a"},{"action":"set_bucket","bucket":"b","size":1,"task":{"action":"run_bucket","bucket":"c"}},{"action":"fill_bucket","bucket":"b"},{"action":"set_bucket","bucket":"c","size":1,"task":{"action":"set_var","name":"passed","value":1}},{"action":"fill_bucket","bucket":"c"},{"action":"run_bucket","bucket":"a"}]},{"when":{"value_equals":{"value":"var(passed)","amount":1}},"then":[{"action":"delete_unit","max":1}]},{"when":{"always":{}},"then":[{"action":"run_bucket","bucket":"c"}]},{"when":{"value_equals":{"value":"var(passed)","amount":1}},"then":[{"action":"delete_unit","max":1}]}]`
	for _, depth := range []string{`"max_bucket_depth":1,`, `"max_bucket_depth":2,`, ``} {
		var diagnostics bytes.Buffer
		model := jsonModel(t, `{`+depth+rules+`}`, &diagnostics)
		if _, ok := model.(*ai.RulesModel); !ok {
			t.Fatalf("model %T, %s", model, diagnostics.String())
		}
		view := &fakeView{units: []ai.UnitView{{InternalID: 73}, {InternalID: 91}, {InternalID: 101}}}
		calls := []orderCall{{Kind: 12, Subject: 73, Clear: true}}
		if depth == "" {
			calls = append(calls, calls[0])
		}
		assertCalls(t, model.DecideOrders(view), view, calls...)
		warnings := strings.Count(diagnostics.String(), "max_bucket_depth")
		if depth == "" && warnings != 0 || depth != "" && warnings != 1 {
			t.Fatalf("depth %s: %q", depth, diagnostics.String())
		}
	}
}

func TestFailedParseIsAtomicAndIsolated(t *testing.T) {
	var diagnostics bytes.Buffer
	model := jsonModel(t, `{"rules":[{"when":{"always":{}},"then":[{"action":"set_var","name":"partial","value":"var(missing)"},{"action":"invalid"}]}]}`, &diagnostics)
	if _, ok := model.(ai.NoopModel); !ok || !strings.Contains(diagnostics.String(), "rule 0") {
		t.Fatalf("model %T, diagnostics %q", model, diagnostics.String())
	}
	if decision := model.DecideOrders(nil); !reflect.DeepEqual(decision, ai.Decision{}) {
		t.Fatalf("partially parsed model executed: %+v", decision)
	}
	diagnostics.Reset()
	valid := jsonModel(t, `{"rules":[]}`, &diagnostics)
	if _, ok := valid.(*ai.RulesModel); !ok || diagnostics.Len() != 0 {
		t.Fatalf("next parse contaminated: %T, %q", valid, diagnostics.String())
	}
}

func TestParseModelEmpty(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	raw := map[string]any{"rules": []any{}}
	model := ai.ParseModel(raw, rng)
	if _, ok := model.(*ai.RulesModel); !ok {
		t.Fatalf("expected RulesModel, got %T", model)
	}
	for _, model := range []ai.Model{model, ai.NoopModel{}} {
		for turn := 0; turn < 2; turn++ {
			model.ApplyUpdate(&fakeView{}, "turn")
			decision := model.DecideOrders(&fakeView{})
			if len(decision.Orders) != 0 || len(decision.Behaviors) != 0 || len(decision.BuildingBehaviors) != 0 {
				t.Fatalf("empty model emitted %+v", decision)
			}
		}
	}
}
