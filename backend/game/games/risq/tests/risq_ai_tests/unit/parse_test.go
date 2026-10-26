package unit

import (
	"fmt"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"io"
	"math"
	"math/rand"
	"testing"
)

func requireNoop(t *testing.T, config string) {
	t.Helper()
	model := jsonModel(t, config, io.Discard)
	if _, ok := model.(ai.NoopModel); !ok {
		t.Fatalf("expected NoopModel, got %T: %s", model, config)
	}
	if decision := model.DecideOrders(nil); len(decision.Orders)+len(decision.Behaviors)+len(decision.BuildingBehaviors) != 0 {
		t.Fatalf("invalid config executed: %+v", decision)
	}
}

var booleanOptions = []struct{ action, key string }{
	{`"action":"set_var","name":"x","value":1`, "persist"},
}

func TestSpaceConditionContracts(t *testing.T) {
	for _, where := range []string{`null`, `[]`, `{"unknown":1}`, `{"vision":5}`, `{"vision":-1}`, `{"vision":1.5}`, `{"vision":null}`, `{"owner":""}`, `{"owner":"friend"}`, `{"all":[null]}`, `{"any":[true]}`, `{"not":null}`, `{"resource_ids":[0]}`, `{"building_ids":[1.5]}`} {
		requireNoop(t, `{"rules":[{"when":{"always":null},"then":[{"action":"set_var","name":"count","value":{"count":{"kind":"spaces","where":`+where+`}}}]}]}`)
	}
	for _, query := range []string{`{"count":null}`, `{"count":[]}`, `{"count":{"kind":"spaces"},"extra":1}`, `{"unknown":{}}`} {
		requireNoop(t, `{"rules":[{"when":{"always":null},"then":[{"action":"set_var","name":"count","value":`+query+`}]}]}`)
	}
}

func TestNonfiniteNumericFields(t *testing.T) {
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		for _, key := range []string{"value"} {
			action := map[string]any{"action": "set_var", "name": "x", "value": float64(1)}
			action[key] = value
			raw := map[string]any{"rules": []any{map[string]any{"when": map[string]any{"always": nil}, "then": []any{action}}}}
			if model := ai.ParseModelWithDiagnostics(raw, rand.New(rand.NewSource(1)), io.Discard); model != (ai.NoopModel{}) {
				t.Fatalf("%s accepted %v: %T", key, value, model)
			}
		}
	}
}

func TestOptionalObjectFields(t *testing.T) {
	for _, value := range []string{`null`, `true`, `1`, `"bad"`, `[]`} {
		requireNoop(t, fmt.Sprintf(`{"rules":[{"when":{"always":null},"then":[{"action":"for_each","source":"units","as":"u","where":%s,"do":[]}]}]}`, value))
	}
}

func TestNumericActionFieldTypes(t *testing.T) {
	for key, action := range map[string]string{
		"max": `"action":"for_each","source":"units","as":"u","do":[]`,
	} {
		for _, value := range []string{`2`, `"1 + 1"`, `{"count":{"kind":"spaces"}}`, `null`, `true`, `[]`, `"1 +"`, `{"unknown":{}}`} {
			t.Run(key+"/"+value, func(t *testing.T) {
				config := fmt.Sprintf(`{"rules":[{"when":{"always":null},"then":[{%s,%q:%s}]}]}`, action, key, value)
				if value == `2` || value == `"1 + 1"` || value == `{"count":{"kind":"spaces"}}` {
					requireModel(t, config)
				} else {
					requireNoop(t, config)
				}
			})
		}
	}
}

func TestMalformedRuleNesting(t *testing.T) {
	for _, config := range []string{`{}`, `{"rules":null}`, `{"rules":{}}`, `{"rules":[null]}`, `{"rules":[{}]}`, `{"rules":[{"when":{},"then":[]}]}`, `{"rules":[{"when":{"always":null},"then":null}]}`, `{"rules":[{"when":{"always":null},"then":[{}]}]}`, `{"rules":[{"when":{"all":[true]},"then":[]}]}`, `{"rules":[{"when":{"not":[]},"then":[]}]}`, `{"rules":[{"when":{"always":null,"any":[]},"then":[]}]}`} {
		requireNoop(t, config)
	}
}

func TestOptionalBooleanTypes(t *testing.T) {
	for _, option := range booleanOptions {
		for _, value := range []string{`true`, `false`, `null`, `1`, `"false"`, `[]`, `{}`} {
			t.Run(option.key+"/"+value, func(t *testing.T) {
				config := fmt.Sprintf(`{"rules":[{"when":{"always":null},"then":[{%s,%q:%s}]}]}`, option.action, option.key, value)
				if value == "true" || value == "false" {
					requireModel(t, config)
				} else {
					requireNoop(t, config)
				}
			})
		}
	}
}

func TestParseModelInvalidRules(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	raw := map[string]any{"rules": "not a list"}
	model := ai.ParseModelWithDiagnostics(raw, rng, io.Discard)
	if _, ok := model.(ai.NoopModel); !ok {
		t.Fatalf("expected NoopModel for invalid rules, got %T", model)
	}
}
