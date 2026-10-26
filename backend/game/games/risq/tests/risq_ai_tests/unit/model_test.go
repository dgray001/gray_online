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
