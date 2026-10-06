package unit

import (
	"encoding/json"
	. "github.com/dgray001/gray_online/game/games/risq/ai"
	"math/rand"
	"testing"
)

const maxNesting, maxLoopSteps = 4, 10000

type decisionView struct{ View }

func (v decisionView) IdleUnits() []UnitView { return []UnitView{{InternalID: 1}} }
func (v decisionView) MoveOrder(u UnitView, target ZoneRef, clear bool) Order {
	return Order{Subjects: []uint64{u.InternalID}, TargetID: int64(target.Space.X), ClearPreviousOrders: clear}
}

func decide(t *testing.T, config string, view View, names ...string) map[string]float64 {
	t.Helper()
	var raw map[string]any
	if err := json.Unmarshal([]byte(config), &raw); err != nil {
		t.Fatal(err)
	}
	var probes []any
	for _, name := range names {
		probes = append(probes, map[string]any{"action": "move", "targets": "coordinate", "coordinate": map[string]any{"x": "var(" + name + ")", "y": float64(0)}})
	}
	raw["rules"] = append(raw["rules"].([]any), map[string]any{"when": map[string]any{"always": map[string]any{}}, "then": probes})
	model, ok := ParseModel(raw, rand.New(rand.NewSource(1))).(*RulesModel)
	if !ok {
		t.Fatalf("config did not load: %s", config)
	}
	decision := model.DecideOrders(decisionView{view})
	if len(decision.Orders) != len(names) {
		t.Fatalf("probe orders=%+v, want %d", decision, len(names))
	}
	values := make(map[string]float64, len(names))
	for i, name := range names {
		values[name] = float64(decision.Orders[i].TargetID)
	}
	return values
}
