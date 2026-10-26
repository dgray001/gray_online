package unit

import (
	"fmt"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"testing"
)

var counterFixtures = []struct {
	name, fields string
	value        float64
}{
	{"value", `,"value":"3 * 4"`, 12},
}

func TestSpaceConditionZeroPointers(t *testing.T) {
	view := &fakeView{}
	called := false
	view.matchingSpaces = func(where ai.SpaceCondition) []ai.Coordinate {
		called = true
		if where.Vision == nil || *where.Vision != 0 || where.PlayerID == nil || *where.PlayerID != 0 || where.UnitCountAtMost == nil || *where.UnitCountAtMost != 0 || where.Not == nil || where.Not.Owner != "enemy" {
			t.Fatalf("lost query fields: %+v", where)
		}
		return nil
	}
	actionsModel(t, `{"action":"set_var","name":"count","value":{"count":{"kind":"spaces","where":{"vision":0,"player_id":0,"unit_count_at_most":0,"not":{"owner":"enemy"}}}}}`).DecideOrders(view)
	if !called {
		t.Fatal("the space condition was never queried")
	}
}

func TestCounterExpressionFailure(t *testing.T) {
	for _, counter := range counterFixtures {
		model := requireModel(t, fmt.Sprintf(`{"rules":[{"when":{%q:{"amount":"1 / (gameVar(turn) - 7)"%s}},"then":[%s]}]}`, counter.name+"_equals", counter.fields, deleteProbe))
		view := counterView()
		assertCalls(t, model.DecideOrders(view), view)
	}
}

func TestCounterComparisonBoundaries(t *testing.T) {
	for _, counter := range counterFixtures {
		for _, comparison := range []string{"at_least", "at_most", "equals"} {
			for _, delta := range []float64{-1, 0, 1} {
				for _, expression := range []bool{false, true} {
					threshold := fmt.Sprint(counter.value + delta)
					if expression {
						threshold = fmt.Sprintf(`"%v + 0"`, counter.value+delta)
					}
					config := fmt.Sprintf(`{"rules":[{"when":{%q:{"amount":%s%s}},"then":[%s]}]}`, counter.name+"_"+comparison, threshold, counter.fields, deleteProbe)
					model, view := requireModel(t, config), counterView()
					var calls []orderCall
					if comparison == "at_least" && delta <= 0 || comparison == "at_most" && delta >= 0 || comparison == "equals" && delta == 0 {
						calls = append(calls, orderCall{Kind: 12, Subject: 73, Clear: true})
					}
					assertCalls(t, model.DecideOrders(view), view, calls...)
				}
			}
		}
	}
}

func counterView() *fakeView {
	return &fakeView{turnNumber: 7, units: []ai.UnitView{{InternalID: 73, UnitID: 1, Kind: ai.UnitEconomic, Type: ai.UnitTypeEconomic}}}
}
