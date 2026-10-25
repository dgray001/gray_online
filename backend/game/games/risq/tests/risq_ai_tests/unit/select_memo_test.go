package unit

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"reflect"
	"testing"

	. "github.com/dgray001/gray_online/game/games/risq/ai"
)

type memoCountingView struct {
	selectTestView
	source_reads int
}

func (v *memoCountingView) MatchingSpaces(condition SpaceCondition) []Coordinate {
	v.source_reads++
	return v.selectTestView.MatchingSpaces(condition)
}

func memoModel(t testing.TB, config string) (*RulesModel, *rand.Rand) {
	t.Helper()
	var raw map[string]any
	if err := json.Unmarshal([]byte(config), &raw); err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewSource(42))
	model, ok := ParseModelWithDiagnostics(raw, rng, io.Discard).(*RulesModel)
	if !ok {
		t.Fatalf("invalid memoization test config: %s", config)
	}
	return model, rng
}

func memoAssignment(actor_lets string, target_lets string, score string) string {
	return rule(fmt.Sprintf(`{"action":"select","actors":{"kind":"units","let":%s,"rank":"0 - var(actor.uid)"},
		"pick":{"kind":"spaces","let":%s},"match":{"score":%q},
		"issue":[{"move":{"to":"space"}}]}`, actor_lets, target_lets, score))
}

func TestSelectMemoizationPreservesAssignmentContext(t *testing.T) {
	units := []UnitView{villager(1, Coordinate{}), villager(2, Coordinate{X: 4})}
	view := selectTestView{loopTestView: loopTestView{units: units, spaces: lineOfSpaces(0, 0, 0)}, idle: units}
	cases := []struct {
		name, actor_lets, target_lets, score, reference string
	}{
		{"target property", `{}`, `{"v":"var(target.x) * 3"}`, "0 - var(target.v)", "0 - var(target.x) * 3"},
		{"actor property", `{"v":"var(actor.unit_id) * 3"}`, `{}`, "0 - var(actor.v)", "0 - var(actor.unit_id) * 3"},
		{"combined lets", `{"a":"var(actor.unit_id)"}`, `{"t":"var(target.x)"}`, "0 - var(actor.a) - var(target.t)", "0 - var(actor.unit_id) - var(target.x)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cached, rng := memoModel(t, memoAssignment(tc.actor_lets, tc.target_lets, tc.score))
			reference, expected_rng := memoModel(t, memoAssignment(`{}`, `{}`, tc.reference))
			got, want := cached.DecideOrders(view), reference.DecideOrders(view)
			if !reflect.DeepEqual(got, want) || rng.Int63() != expected_rng.Int63() {
				t.Fatalf("cached decision %+v differs from inline decision %+v", got, want)
			}
		})
	}
}

func TestSelectMemoizationCachesNestedRandomness(t *testing.T) {
	units := []UnitView{villager(1, Coordinate{}), villager(2, Coordinate{})}
	view := selectTestView{loopTestView: loopTestView{units: units, spaces: lineOfSpaces(1)}, idle: units}
	cases := []struct {
		name, lets string
		draws      int
		tied       bool
	}{
		{"alias", `{"r":"random()","value":"var(target.r)"}`, 1, false},
		{"alias chain", `{"r":"random()","a":"var(target.r)","value":"var(target.a)"}`, 1, false},
		{"aggregate", `{"value":{"sum":{"kind":"units","of":"random()"}}}`, 2, false},
		{"aggregate alias", `{"r":"random()","value":{"sum":{"kind":"units","of":"var(target.r)"}}}`, 1, false},
		{"aggregate filter", `{"value":{"count":{"kind":"units","where":{"if":"random() > -1"}}}}`, 2, false},
		{"implicit ties", `{"value":{"count":{"kind":"units","limit":1}}}`, 1, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := rule(fmt.Sprintf(`{"action":"select","pick":{"kind":"spaces","let":%s,"where":{"if":"var(target.value) > -1"},"rank":"var(target.value)"},"into":{"set":"chosen"}}`, tc.lets))
			model, rng := memoModel(t, config)
			model.DecideOrders(view)
			expected := rand.New(rand.NewSource(42))
			for range tc.draws {
				if tc.tied {
					expected.Intn(2)
				} else {
					expected.Float64()
				}
			}
			if rng.Int63() != expected.Int63() {
				t.Fatal("memoization changed the RNG sequence")
			}
		})
	}
}

func TestSelectMemoizationReducesSourceScans(t *testing.T) {
	view := &memoCountingView{selectTestView: selectTestView{loopTestView: loopTestView{spaces: lineOfSpaces(make([]float64, 100)...)}}}
	config := rule(`{"action":"select","pick":{"kind":"spaces","let":{"all":{"count":{"kind":"spaces"}}},
		"where":{"if":"var(target.all)"},"rank":"var(target.all)"},"into":{"set":"chosen"}}`)
	model, _ := memoModel(t, config)
	model.DecideOrders(view)
	if view.source_reads != 101 {
		t.Fatalf("enumerated spaces %d times, want 101 instead of 201", view.source_reads)
	}
}

func TestSelectMemoizationDoesNotCacheLoopBodyValues(t *testing.T) {
	view := selectTestView{loopTestView: loopTestView{units: []UnitView{villager(1, Coordinate{}), villager(2, Coordinate{})}}}
	config := rule(`{"action":"set_var","name":"counter","value":0},{"action":"set_var","name":"observed","value":0},
		{"action":"for_each","select":{"kind":"units","let":{"value":"var(counter)"}},"as":"u","do":[
		{"action":"set_var","name":"observed","value":"10 * var(observed) + var(u.value)"},
		{"action":"set_var","name":"counter","value":"var(counter) + 1"},
		{"action":"set_var","name":"observed","value":"10 * var(observed) + var(u.value)"}]}`)
	if got := decide(t, config, view, "observed")["observed"]; got != 112 {
		t.Fatalf("observed %v, want 112 from live loop variables", got)
	}
}

func BenchmarkSelectLetMemoization(b *testing.B) {
	query := `{"count":{"kind":"spaces"}}`
	baseline := rule(fmt.Sprintf(`{"action":"select","pick":{"kind":"spaces","where":{"if":%s},"rank":%s},"into":{"set":"chosen"}}`, query, query))
	memoized := rule(fmt.Sprintf(`{"action":"select","pick":{"kind":"spaces","let":{"all":%s},"where":{"if":"var(target.all)"},"rank":"var(target.all)"},"into":{"set":"chosen"}}`, query))
	for name, config := range map[string]string{"uncached": baseline, "memoized": memoized} {
		b.Run(name, func(b *testing.B) {
			model, _ := memoModel(b, config)
			view := selectTestView{loopTestView: loopTestView{spaces: lineOfSpaces(make([]float64, 100)...)}}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				model.DecideOrders(view)
			}
		})
	}
}
