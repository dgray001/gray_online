package unit

import (
	"fmt"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"io"
	"math/rand"
	"reflect"
	"testing"
)

func TestBucketScenario(t *testing.T) {
	model := requireModel(t, bucketScenario)
	for turn := 1; turn <= 3; turn++ {
		view := economyView(turn)
		worker := view.units[1]
		worker.InternalID = 96
		view.units = append(view.units[:2], worker, view.units[4])
		var expected []namedCall
		switch turn {
		case 1:
			expected = []namedCall{{"drained reserved worker", orderCall{Kind: 1, Subject: 91, Place: ai.ZoneRef{Space: ai.Coordinate{X: 2}}, Clear: true}}, {"first retained worker", orderCall{Kind: 2, Subject: 92, Target: 401, Place: ai.ZoneRef{Space: ai.Coordinate{X: 3}}, Clear: true}}, {"second retained worker", orderCall{Kind: 2, Subject: 96, Target: 401, Place: ai.ZoneRef{Space: ai.Coordinate{X: 3}}, Clear: true}}, {"bucket membership", orderCall{Kind: 1, Subject: 99, Place: ai.ZoneRef{Space: ai.Coordinate{X: 21}}, Clear: true}}}
		case 2:
			expected = []namedCall{{"empty retains task", orderCall{Kind: 1, Subject: 91, Place: ai.ZoneRef{Space: ai.Coordinate{X: 2}}, Clear: true}}, {"shrink retains lowest id", orderCall{Kind: 1, Subject: 92, Place: ai.ZoneRef{Space: ai.Coordinate{X: 3}}, Clear: true}}, {"bucket membership", orderCall{Kind: 1, Subject: 99, Place: ai.ZoneRef{Space: ai.Coordinate{X: 11}}, Clear: true}}}
		case 3:
			expected = []namedCall{{"drain any source", orderCall{Kind: 1, Subject: 92, Place: ai.ZoneRef{Space: ai.Coordinate{X: 2}}, Clear: true}}, {"bucket membership", orderCall{Kind: 1, Subject: 99, Place: ai.ZoneRef{Space: ai.Coordinate{X: 1}}, Clear: true}}}
		}
		assertScenarioCalls(t, fmt.Sprint("buckets turn ", turn), model.DecideOrders(view), view, expected)
	}
}

func TestArmyStrikeCharacterization(t *testing.T) {
	var models []ai.Model
	for _, strike := range []string{"", `,"strike":0`, `,"strike":1`, `,"strike":100`} {
		models = append(models, requireModel(t, `{"rules":[{"when":{"always":{}},"then":[{"action":"army","launch":1,"retreat":0`+strike+`}]}]}`))
	}
	for turn := 0; turn < 4; turn++ {
		var want ai.Decision
		var calls []orderCall
		for index, model := range models {
			view := populatedView()
			if turn == 1 {
				view.visibleEnemyUnits = nil
			}
			if turn == 2 {
				view.units = view.units[:1]
			}
			decision := model.DecideOrders(view)
			if index == 0 {
				want, calls = decision, view.calls
			}
			if !reflect.DeepEqual(decision, want) || !reflect.DeepEqual(view.calls, calls) {
				t.Fatalf("turn %d, model %d: %+v/%+v, want %+v/%+v", turn, index, decision, view.calls, want, calls)
			}
		}
	}
}

func TestBucketPersistencePruningAndFullness(t *testing.T) {
	for _, full := range []bool{false, true} {
		config := fmt.Sprintf(`{"rules":[{"when":{"turn_equals":{"amount":1}},"then":[{"action":"set_bucket","bucket":"team","size":2,"task":{"action":"move","targets":"coordinate","coordinate":{"x":2,"y":-1}}},{"action":"fill_bucket","bucket":"team"}]},{"when":{"always":{}},"then":[{"action":"run_bucket","bucket":"team","when_full":%v}]}]}`, full)
		model := requireModel(t, config)
		first := &fakeView{turnNumber: 1, units: []ai.UnitView{{InternalID: 73}, {InternalID: 91}, {InternalID: 101}}}
		place := ai.ZoneRef{Space: ai.Coordinate{X: 2, Y: -1}}
		assertCalls(t, model.DecideOrders(first), first, orderCall{Kind: 1, Subject: 73, Place: place, Clear: true}, orderCall{Kind: 1, Subject: 91, Place: place, Clear: true})
		second := &fakeView{turnNumber: 2, units: []ai.UnitView{{InternalID: 91}, {InternalID: 101}}}
		var calls []orderCall
		if !full {
			calls = append(calls, orderCall{Kind: 1, Subject: 91, Place: place, Clear: true})
		}
		assertCalls(t, model.DecideOrders(second), second, calls...)
		empty := &fakeView{turnNumber: 3}
		assertCalls(t, model.DecideOrders(empty), empty)
		other, fresh := requireModel(t, config), &fakeView{turnNumber: 2, units: []ai.UnitView{{InternalID: 91}}}
		assertCalls(t, other.DecideOrders(fresh), fresh)
	}
}

func TestVariableRefreshAndApplyUpdate(t *testing.T) {
	config := `{"rules":[{"when":{"turn_equals":{"amount":1}},"then":[{"action":"set_var","name":"shared","value":5,"persist":true},{"action":"set_var","name":"shared","value":9},{"action":"set_var","name":"temporary","value":3}]},{"when":{"always":{}},"then":[{"action":"move","targets":"coordinate","coordinate":{"x":"var(shared)","y":"var(temporary)"}}]}]}`
	model := requireModel(t, config)
	for index, target := range []ai.Coordinate{{X: 9, Y: 3}, {X: 5}, {X: 5}} {
		view := &fakeView{turnNumber: index + 1, units: []ai.UnitView{{InternalID: 73}}}
		model.ApplyUpdate(view, "turn")
		model.ApplyUpdate(view, "turn")
		assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: 1, Subject: 73, Place: ai.ZoneRef{Space: target}, Clear: true})
	}
	other := requireModel(t, config)
	view := &fakeView{turnNumber: 2, units: []ai.UnitView{{InternalID: 73}}}
	assertCalls(t, other.DecideOrders(view), view, orderCall{Kind: 1, Subject: 73, Clear: true})
}

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
	model := ai.ParseModelWithDiagnostics(raw, rng, io.Discard)
	if _, ok := model.(*ai.RulesModel); !ok {
		t.Fatalf("expected RulesModel, got %T", model)
	}

	view := &fakeView{}
	// First turn
	model.DecideOrders(view)
	// Second turn, testing isolation/persistence conceptually
	model.DecideOrders(view)
}
