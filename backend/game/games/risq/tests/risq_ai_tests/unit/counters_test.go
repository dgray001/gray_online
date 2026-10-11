package unit

import (
	"fmt"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"io"
	"math/rand"
	"testing"
)

var counterFixtures = []struct {
	name, fields string
	value        float64
}{
	{"turn", "", 7}, {"map_size", "", 4}, {"num_players", "", 3}, {"enemies_found", "", 2}, {"land", "", 6}, {"score_lead", "", -3}, {"population_headroom", "", 5},
	{"resource", `,"category":"food"`, 13}, {"resource", `,"category":"wood"`, 17}, {"resource", `,"category":"stone"`, 19}, {"resource", `,"category":"gold"`, 23},
	{"resource_remaining", `,"category":"wood"`, 68}, {"resource_remaining", `,"category":"food"`, 41},
	{"population", "", 3}, {"population", `,"unit_ids":[2]`, 1}, {"population", `,"unit_types":["economic"]`, 2}, {"population", `,"unit_ids":[1],"unit_types":["military"]`, 3},
	{"idle_units", "", 3}, {"enemy_units_visible", `,"unit_types":["military"]`, 1}, {"enemy_units_unidentified", "", 4}, {"enemy_buildings_known", `,"building_ids":[2]`, 1},
	{"building_count", "", 1}, {"building_count", `,"state":"complete"`, 1}, {"building_count", `,"state":"damaged"`, 0}, {"building_count", `,"state":"depleted"`, 0},
	{"foundation_count", "", 2}, {"foundation_count", `,"without_builders":true`, 1}, {"foundation_space_x", "", -2}, {"foundation_space_y", "", 3},
	{"bucket_size", `,"bucket":"missing"`, 0}, {"economic_producers", "", 1}, {"available_gatherers", "", 1},
	{"count_spaces", "", 0}, {"closest_spaces", "", 0}, {"base_spaces", "", 1}, {"value", `,"value":"3 * 4"`, 12},
}

func TestSpaceConditionZeroPointers(t *testing.T) {
	view := checkedView("IdleUnits", "EligibleUnits", "CountSpaces")
	view.countSpaces = func(where ai.SpaceCondition, include func(ai.Coordinate) bool) int {
		if where.Vision == nil || *where.Vision != 0 || where.PlayerID == nil || *where.PlayerID != 0 || where.UnitCountAtMost == nil || *where.UnitCountAtMost != 0 || where.Not == nil || where.Not.Owner != "enemy" || !include(ai.Coordinate{X: 2}) {
			t.Fatalf("lost query fields or predicate: %+v", where)
		}
		return 0
	}
	model := actionsModel(t, `{"action":"set_var","name":"count","value":{"count_spaces":{"where":{"vision":0,"player_id":0,"unit_count_at_most":0,"not":{"owner":"enemy"}}}}},{"action":"move","targets":"coordinate","coordinate":{"x":"var(count)","y":0}}`)
	assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: 1, Subject: 91, Clear: true})
}

func TestCounterExpressionFailure(t *testing.T) {
	for _, counter := range counterFixtures {
		model := requireModel(t, fmt.Sprintf(`{"rules":[{"when":{%q:{"amount":"1 / (turn - 7)"%s}},"then":[{"action":"delete_unit"}]}]}`, counter.name+"_equals", counter.fields))
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
					config := fmt.Sprintf(`{"rules":[{"when":{%q:{"amount":%s%s}},"then":[{"action":"delete_unit","max":1}]}]}`, counter.name+"_"+comparison, threshold, counter.fields)
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
	count, garrison := 5, uint64(800)
	return &fakeView{turnNumber: 7, mapSize: 4, numPlayers: 3, enemiesFound: 2, ownedSpaces: 6, score: 11, bestEnemyScore: 14, popCurrent: 3, popLimit: 8,
		units:               []ai.UnitView{{InternalID: 73, UnitID: 1, Kind: ai.UnitEconomic, Type: ai.UnitTypeEconomic}, {InternalID: 91, UnitID: 2, Kind: ai.UnitMilitary, Type: ai.UnitTypeInfantry, Location: ai.ZoneRef{Space: ai.Coordinate{X: 1}}}, {InternalID: 101, UnitID: 1, Kind: ai.UnitEconomic, Type: ai.UnitTypeEconomic, GarrisonedIn: &garrison}},
		resources:           map[ai.ResourceCategory]float64{ai.ResourceFood: 13, ai.ResourceWood: 17, ai.ResourceStone: 19, ai.ResourceGold: 23},
		buildings:           []ai.BuildingView{{InternalID: 800, BuildingID: 1, Producibles: []ai.Producible{{ID: 1, Kind: ai.ProducibleUnit, UnitType: ai.UnitTypeEconomic}, {ID: 2, Kind: ai.ProducibleUnit, UnitType: ai.UnitTypeEconomic}}}},
		visibleEnemyUnits:   []ai.UnitView{{InternalID: 201, UnitID: 2, Kind: ai.UnitMilitary, Type: ai.UnitTypeInfantry}},
		knownEnemyBuildings: []ai.BuildingView{{InternalID: 301, BuildingID: 2}},
		knownResources:      []ai.ResourceView{{Category: ai.ResourceWood, AmountLeft: 31}, {Category: ai.ResourceWood, AmountLeft: 37}, {Category: ai.ResourceFood, AmountLeft: 41}},
		foundations:         []ai.FoundationView{{BuildingID: 2, Planned: true, Location: ai.ZoneRef{Space: ai.Coordinate{X: -2, Y: 3}}}, {BuildingID: 3, Builders: 1, Location: ai.ZoneRef{Space: ai.Coordinate{X: 4, Y: -1}}}},
		allSpaces:           []ai.SpaceInfo{{UnitCount: &count}},
	}
}

func runCounterCondition(t *testing.T, condition map[string]any, view ai.View) []ai.Order {
	rng := rand.New(rand.NewSource(1))
	raw := map[string]any{
		"rules": []any{
			map[string]any{
				"when": condition,
				"then": []any{
					map[string]any{
						"action": "delete_unit", // produces a dummy order to verify execution
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

func TestCounterResource(t *testing.T) {
	view := &fakeView{
		units: []ai.UnitView{{InternalID: 1}},
		resources: map[ai.ResourceCategory]float64{
			ai.ResourceWood: 150.0,
		},
	}

	condEqual := map[string]any{"resource_equals": map[string]any{"category": "wood", "amount": 150.0}}
	condMore := map[string]any{"resource_at_least": map[string]any{"category": "wood", "amount": 100.0}}
	condLess := map[string]any{"resource_at_most": map[string]any{"category": "wood", "amount": 50.0}}

	if len(runCounterCondition(t, condEqual, view)) == 0 {
		t.Fatal("expected resource_equals 150.0 to be true")
	}
	if len(runCounterCondition(t, condMore, view)) == 0 {
		t.Fatal("expected resource_at_least 100.0 to be true")
	}
	if len(runCounterCondition(t, condLess, view)) != 0 {
		t.Fatal("expected resource_at_most 50.0 to be false")
	}
}

func TestCounterPopulation(t *testing.T) {
	view := &fakeView{
		units: []ai.UnitView{
			{InternalID: 1, Type: ai.UnitTypeEconomic},
			{InternalID: 2, Type: ai.UnitTypeInfantry},
		},
	}

	condEqual := map[string]any{"population_equals": map[string]any{"amount": 2.0}}
	condLess := map[string]any{"population_at_most": map[string]any{"amount": 1.0}}

	if len(runCounterCondition(t, condEqual, view)) == 0 {
		t.Fatal("expected population_equals 2.0 to be true")
	}
	if len(runCounterCondition(t, condLess, view)) != 0 {
		t.Fatal("expected population_at_most 1.0 to be false")
	}
}

func TestCounterIdleUnits(t *testing.T) {
	view := &fakeView{
		units: []ai.UnitView{
			{InternalID: 1, Type: ai.UnitTypeEconomic},
			{InternalID: 2, Type: ai.UnitTypeEconomic},
		},
	}
	// fakeView's IdleUnits returns all units currently

	condEqual := map[string]any{"idle_units_equals": map[string]any{"amount": 2.0}}
	condLess := map[string]any{"idle_units_at_most": map[string]any{"amount": 1.0}}

	if len(runCounterCondition(t, condEqual, view)) == 0 {
		t.Fatal("expected idle_units_equals 2.0 to be true")
	}
	if len(runCounterCondition(t, condLess, view)) != 0 {
		t.Fatal("expected idle_units_at_most 1.0 to be false")
	}
}
