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
	{`"action":"set_var","name":"x","value":1`, "persist"}, {`"action":"produce"`, "prioritize"},
	{`"action":"attack"`, "resume"}, {`"action":"move","targets":"home"`, "together"},
	{`"action":"run_bucket","bucket":"x"`, "when_full"}, {`"action":"drain_bucket","from":[],"to":"x"`, "allow_reserved"},
	{`"action":"delete_unit"`, "exclude_buckets"}, {`"action":"build","building_id":1`, "reuse_unbuilt_foundation"},
	{`"action":"move","targets":"closest_spaces"`, "exclude_reference"}, {`"action":"set_unit_behavior"`, "attack_back"},
	{`"action":"set_unit_behavior"`, "interrupt_current"}, {`"action":"set_building_behavior"`, "auto_attack"},
}

func TestDrainSourceTypes(t *testing.T) {
	for _, c := range []struct {
		value string
		valid bool
	}{{`"workers"`, true}, {`"any"`, true}, {`[]`, true}, {`["a","b"]`, true}, {`""`, false}, {`[""]`, false}, {`["a",null]`, false}, {`["a",1]`, false}, {`null`, false}} {
		config := `{"rules":[{"when":{"always":null},"then":[{"action":"drain_bucket","to":"target","from":` + c.value + `}]}]}`
		if c.valid {
			requireModel(t, config)
		} else {
			requireNoop(t, config)
		}
	}
}

func TestSpaceConditionContracts(t *testing.T) {
	for _, condition := range []string{`{"bucket_full":{"bucket":""}}`, `{"bucket_size_equals":{"bucket":"","amount":0}}`, `{"population_equals":{"in_bucket":null,"amount":0}}`, `{"idle_units_equals":{"in_bucket":false,"amount":0}}`, `{"building_count_equals":{"state":1,"amount":0}}`, `{"foundation_count_equals":{"without_builders":"true","amount":0}}`, `{"foundation_space_x_equals":{"without_builders":null,"amount":0}}`} {
		requireNoop(t, `{"rules":[{"when":`+condition+`,"then":[]}]}`)
	}
	for _, where := range []string{`null`, `[]`, `{"unknown":1}`, `{"vision":5}`, `{"vision":-1}`, `{"vision":1.5}`, `{"vision":null}`, `{"owner":""}`, `{"owner":"friend"}`, `{"all":[null]}`, `{"any":[true]}`, `{"not":null}`, `{"resource_ids":[0]}`, `{"building_ids":[1.5]}`} {
		requireNoop(t, `{"rules":[{"when":{"always":null},"then":[{"action":"set_var","name":"count","value":{"count_spaces":{"where":`+where+`}}}]}]}`)
	}
	for _, query := range []string{`{"count_spaces":null}`, `{"closest_spaces":[]}`, `{"count_spaces":{},"extra":1}`, `{"unknown":{}}`} {
		requireNoop(t, `{"rules":[{"when":{"always":null},"then":[{"action":"set_var","name":"count","value":`+query+`}]}]}`)
	}
}

func TestNonfiniteNumericFields(t *testing.T) {
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		for _, key := range []string{"value", "max", "queue", "weight", "move_penalty", "depth", "size", "reserve", "launch", "retreat", "defend_radius", "strike", "score", "min_score", "min_distance"} {
			action := map[string]any{"action": "set_var", "name": "x", "value": float64(1)}
			action[key] = value
			raw := map[string]any{"rules": []any{map[string]any{"when": map[string]any{"always": nil}, "then": []any{action}}}}
			if model := ai.ParseModelWithDiagnostics(raw, rand.New(rand.NewSource(1)), io.Discard); model != (ai.NoopModel{}) {
				t.Fatalf("%s accepted %v: %T", key, value, model)
			}
		}
	}
}

func TestNearFilterTypes(t *testing.T) {
	for _, value := range []string{`0`, `2`, `-1`, `1.5`, `null`, `"2"`, `"1 + 1"`, `true`, `[]`, `{}`} {
		config := fmt.Sprintf(`{"rules":[{"when":{"population_equals":{"amount":1,"from":"home","within":%s}},"then":[]}]}`, value)
		if value == "0" || value == "2" {
			requireModel(t, config)
		} else {
			requireNoop(t, config)
		}
	}
	requireNoop(t, `{"rules":[{"when":{"population_equals":{"amount":1,"from":"home"}},"then":[]}]}`)
}

func TestPickerOptionsAndOutputNames(t *testing.T) {
	for _, options := range []string{`[]`, `null`, `true`, `[null]`, `[{}]`, `[{"targets":null}]`, `[{"targets":"unknown"}]`, `[{"targets":"home","together":null}]`, `[{"targets":"home","spread":{"overflow":null}}]`} {
		requireNoop(t, `{"rules":[{"when":{"always":null},"then":[{"action":"select_space","options":`+options+`}]}]}`)
	}
	requireNoop(t, `{"rules":[{"when":{"always":null},"then":[{"action":"select_space","targets":"home","options":[{"targets":"home"}]}]}]}`)
	for _, key := range []string{"var", "var_x", "var_y", "var_score"} {
		for _, name := range []string{`null`, `1`, `""`, `"1bad"`, `"bad-name"`} {
			requireNoop(t, fmt.Sprintf(`{"rules":[{"when":{"always":null},"then":[{"action":"select_space","targets":"home",%q:%s}]}]}`, key, name))
		}
	}
}

func TestOptionalStringFields(t *testing.T) {
	for _, c := range []struct{ action, key, value string }{{`"action":"attack"`, "target", `"military"`}, {`"action":"attack"`, "targets", `"enemy_units"`}, {`"action":"set_unit_behavior"`, "stance", `"passive"`}, {`"action":"explore"`, "anchor", `"self"`}, {`"action":"attack","targets":"enemy_units"`, "order", `"unit"`}, {`"action":"fill_bucket"`, "bucket", `"workers"`}, {`"action":"delete_unit"`, "in_bucket", `"workers"`}, {`"action":"drain_bucket","from":[]`, "to", `"workers"`}} {
		for _, value := range []string{c.value, `null`, `""`, `1`, `true`, `[]`, `{}`} {
			t.Run(c.key+"/"+value, func(t *testing.T) {
				config := fmt.Sprintf(`{"rules":[{"when":{"always":null},"then":[{%s,%q:%s}]}]}`, c.action, c.key, value)
				if value == c.value {
					requireModel(t, config)
				} else {
					requireNoop(t, config)
				}
			})
		}
	}
}

func TestOptionalObjectFields(t *testing.T) {
	for _, c := range []struct{ action, key string }{{`"action":"delete_unit"`, "unit_when"}, {`"action":"move","targets":"enemy_units"`, "spread"}, {`"action":"move","targets":"closest_spaces"`, "where"}, {`"action":"build","building_id":2`, "site"}} {
		for _, value := range []string{`null`, `true`, `1`, `"bad"`, `[]`} {
			requireNoop(t, fmt.Sprintf(`{"rules":[{"when":{"always":null},"then":[{%s,%q:%s}]}]}`, c.action, c.key, value))
		}
	}
}

func TestNumericActionFieldTypes(t *testing.T) {
	for key, action := range map[string]string{
		"max":   `"action":"for_each","source":"units","as":"u","do":[]`,
		"queue": `"action":"create","unit_id":1`, "depth": `"action":"produce"`,
		"weight": `"action":"gather"`, "move_penalty": `"action":"gather"`,
		"size":    `"action":"set_bucket","bucket":"x","task":{"action":"set_var","name":"x","value":1}`,
		"reserve": `"action":"hire","unit_id":1`,
		"launch":  `"action":"army"`, "retreat": `"action":"army"`,
		"defend_radius": `"action":"army"`, "strike": `"action":"army"`,
		"score":        `"action":"build","building_id":1,"targets":"build_sites"`,
		"min_score":    `"action":"build","building_id":1,"targets":"build_sites","score":1`,
		"min_distance": `"action":"explore"`,
	} {
		for _, value := range []string{`2`, `"1 + 1"`, `{"count_spaces":{}}`, `{"closest_spaces":{}}`, `null`, `true`, `[]`, `"1 +"`, `{"unknown":{}}`} {
			t.Run(key+"/"+value, func(t *testing.T) {
				config := fmt.Sprintf(`{"rules":[{"when":{"always":null},"then":[{%s,%q:%s}]}]}`, action, key, value)
				if value == `2` || value == `"1 + 1"` || value == `{"count_spaces":{}}` || value == `{"closest_spaces":{}}` {
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
	requireNoop(t, `{"rules":[{"when":{"always":{}},"then":[{"action":"move","targets":"home","distribute":{}}]}]}`)
}

func TestMaxBucketDepthTypes(t *testing.T) {
	for _, value := range []string{`1`, `2`, `0`, `-1`, `1.5`, `null`, `"3"`, `true`, `[]`, `{}`} {
		config := fmt.Sprintf(`{"max_bucket_depth":%s,"rules":[]}`, value)
		if value == "1" || value == "2" {
			requireModel(t, config)
		} else {
			requireNoop(t, config)
		}
	}
}

func TestResourceQueueCost(t *testing.T) {
	for _, cost := range []string{`{}`, `{"food":0}`, `{"wood":1.25,"gold":2}`, `null`, `[]`, `{"food":-1}`, `{"food":null}`, `{"food":"1"}`, `{"food":true}`, `{"unknown":1}`} {
		config := fmt.Sprintf(`{"rules":[{"when":{"always":{}},"then":[{"action":"add_q","type":"resource","cost":%s}]}]}`, cost)
		if cost == "{}" || cost == `{"food":0}` || cost == `{"wood":1.25,"gold":2}` {
			requireModel(t, config)
		} else {
			requireNoop(t, config)
		}
	}
}

func TestStringListOptions(t *testing.T) {
	for _, c := range []struct{ action, key, value string }{
		{`"action":"delete_unit"`, "unit_types", `"economic"`}, {`"action":"attack","targets":"enemy_units"`, "target_unit_types", `"military"`},
		{`"action":"gather","category":"wood"`, "eligible", `"gather"`}, {`"action":"unit_stop"`, "orders", `"move"`},
		{`"action":"building_stop"`, "orders", `"create"`}, {`"action":"set_unit_behavior"`, "target_priority", `"building"`},
	} {
		for _, value := range []string{`[]`, `[` + c.value + `]`, `null`, `"economic"`, `[null]`, `[1]`, `[true]`, `["unknown"]`, `{}`} {
			t.Run(c.key+"/"+value, func(t *testing.T) {
				config := fmt.Sprintf(`{"rules":[{"when":{"always":{}},"then":[{%s,%q:%s}]}]}`, c.action, c.key, value)
				if value == "[]" || value == "["+c.value+"]" {
					requireModel(t, config)
				} else {
					requireNoop(t, config)
				}
			})
		}
	}
}

func TestIDFilterTypes(t *testing.T) {
	for _, c := range []struct{ action, key string }{
		{`"action":"delete_unit"`, "unit_ids"}, {`"action":"delete_building"`, "building_ids"}, {`"action":"army"`, "assault_unit_ids"},
		{`"action":"move","targets":"enemy_units"`, "target_unit_ids"}, {`"action":"move","targets":"enemy_buildings"`, "target_building_ids"},
	} {
		for _, value := range []string{`[]`, `[1,1,4294967295]`, `null`, `1`, `[0]`, `[-1]`, `[1.5]`, `[4294967296]`, `[null]`, `["1"]`} {
			t.Run(c.key+"/"+value, func(t *testing.T) {
				config := fmt.Sprintf(`{"rules":[{"when":{"always":{}},"then":[{%s,%q:%s}]}]}`, c.action, c.key, value)
				if value == "[]" || value == "[1,1,4294967295]" {
					requireModel(t, config)
				} else {
					requireNoop(t, config)
				}
			})
		}
	}
}

func TestConfigIDBoundaries(t *testing.T) {
	for _, c := range []struct{ action, key string }{
		{`"action":"create"`, "unit_id"}, {`"action":"hire"`, "unit_id"}, {`"action":"research"`, "tech_id"}, {`"action":"build"`, "building_id"}, {`"action":"add_q","type":"unit"`, "id"},
	} {
		for _, value := range []string{`1`, `4294967295`, `0`, `-1`, `1.5`, `4294967296`, `null`, `"1"`, `true`, `[]`, `{}`} {
			t.Run(c.key+"/"+value, func(t *testing.T) {
				config := fmt.Sprintf(`{"rules":[{"when":{"always":{}},"then":[{%s,%q:%s}]}]}`, c.action, c.key, value)
				if value == "1" || value == "4294967295" {
					requireModel(t, config)
				} else {
					requireNoop(t, config)
				}
			})
		}
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
