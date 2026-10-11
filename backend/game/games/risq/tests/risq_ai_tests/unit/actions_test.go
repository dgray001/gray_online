package unit

import (
	"fmt"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"io"
	"math/rand"
	"reflect"
	"testing"
)

var actionFixtures = []string{
	`{"action":"gather","category":"food"}`, `{"action":"gather"}`,
	`{"action":"create","unit_id":1}`, `{"action":"research","tech_id":1}`, `{"action":"build","building_id":1}`,
	`{"action":"build_foundations"}`, `{"action":"cancel_foundations"}`, `{"action":"renew"}`, `{"action":"repair"}`,
	`{"action":"hire","unit_id":1}`, `{"action":"explore"}`, `{"action":"move","targets":"coordinate","coordinate":{"x":-2,"y":3}}`,
	`{"action":"attack"}`, `{"action":"attack_space"}`, `{"action":"attack_zone"}`, `{"action":"garrison"}`, `{"action":"ungarrison"}`,
	`{"action":"delete_unit"}`, `{"action":"delete_building"}`, `{"action":"building_attack"}`,
	`{"action":"set_unit_behavior","attack_back":false}`, `{"action":"set_building_behavior","auto_attack":false}`,
	`{"action":"unit_stop"}`, `{"action":"building_stop"}`, `{"action":"createNextInQ"}`, `{"action":"researchNextInQ"}`, `{"action":"buildNextInQ"}`, `{"action":"produce"}`,
	`{"action":"add_q","type":"unit","id":1}`, `{"action":"add_q","type":"building","id":1}`, `{"action":"add_q","type":"technology","id":1}`, `{"action":"add_q","type":"resource","cost":{}}`,
	`{"action":"set_var","name":"example","value":2}`, `{"action":"select_space","targets":"own_buildings","var":"selected"}`,
	`{"action":"set_bucket","bucket":"workers","size":2,"task":{"action":"repair"}}`, `{"action":"fill_bucket","bucket":"workers"}`, `{"action":"run_bucket","bucket":"workers"}`,
	`{"action":"drain_bucket","from":[],"to":"workers"}`, `{"action":"empty_bucket","bucket":"workers"}`, `{"action":"army"}`,
}

var queueCases = []struct {
	actions string
	call    orderCall
}{
	{`{"action":"add_q","type":"unit","id":10},{"action":"createNextInQ"}`, orderCall{Kind: 13, Subject: 800, Item: 10}},
	{`{"action":"add_q","type":"technology","id":30},{"action":"researchNextInQ"}`, orderCall{Kind: 14, Subject: 801, Item: 30}},
	{`{"action":"add_q","type":"building","id":20},{"action":"buildNextInQ"}`, orderCall{Kind: 3, Subject: 73, Item: 20, Place: ai.ZoneRef{Space: ai.Coordinate{X: 1, Y: 1}}, Clear: true}},
	{`{"action":"add_q","type":"unit","id":11},{"action":"add_q","type":"unit","id":10},{"action":"createNextInQ","depth":1}`, orderCall{Kind: 13, Subject: 800, Item: 10}},
	{`{"action":"add_q","type":"unit","id":99},{"action":"add_q","type":"unit","id":10},{"action":"createNextInQ","depth":2}`, orderCall{Kind: 13, Subject: 800, Item: 10}},
	{`{"action":"add_q","type":"unit","id":10,"weight":2},{"action":"add_q","type":"unit","id":12,"weight":2},{"action":"produce","prioritize":true,"depth":2}`, orderCall{Kind: 13, Subject: 800, Item: 10}},
	{`{"action":"add_q","type":"unit","id":10,"weight":2},{"action":"add_q","type":"unit","id":12,"weight":3},{"action":"produce","prioritize":true,"depth":2}`, orderCall{Kind: 13, Subject: 800, Item: 12}},
	{`{"action":"add_q","type":"unit","id":10,"weight":2},{"action":"add_q","type":"unit","id":12,"weight":3},{"action":"produce","depth":2}`, orderCall{Kind: 13, Subject: 800, Item: 10}},
}

func TestQueueSelection(t *testing.T) {
	for _, c := range queueCases {
		model, view := requireModel(t, `{"rules":[{"when":{"always":{}},"then":[`+c.actions+`]}]}`), queueView()
		assertCalls(t, model.DecideOrders(view), view, c.call)
	}
	for _, actions := range []string{
		`{"action":"add_q","type":"resource","cost":{"food":5}},{"action":"produce"}`,
		`{"action":"add_q","type":"unit","id":10,"weight":2},{"action":"produce","weight":2}`,
		`{"action":"add_q","type":"unit","id":99},{"action":"add_q","type":"unit","id":10},{"action":"createNextInQ","depth":1}`,
	} {
		model, view := requireModel(t, `{"rules":[{"when":{"always":{}},"then":[`+actions+`]}]}`), queueView()
		assertCalls(t, model.DecideOrders(view), view)
	}
}

var constructorCases = []struct {
	action string
	call   orderCall
}{
	{`"action":"gather","category":"wood","max":1`, orderCall{Kind: 2, Subject: 73, Target: 401, Place: ai.ZoneRef{Space: ai.Coordinate{X: 2}}, Clear: true}},
	{`"action":"build_foundations","max":1`, orderCall{Kind: 3, Subject: 73, Item: 2, Place: ai.ZoneRef{Space: ai.Coordinate{X: 3}}, Clear: true}},
	{`"action":"cancel_foundations"`, orderCall{Kind: 18, Item: 2, Place: ai.ZoneRef{Space: ai.Coordinate{X: 3}}}},
	{`"action":"renew","max":1`, orderCall{Kind: 5, Subject: 73, Target: 800, Clear: true}}, {`"action":"repair","max":1`, orderCall{Kind: 4, Subject: 73, Target: 800, Clear: true}},
	{`"action":"hire","unit_id":2,"reserve":2,"max":1`, orderCall{Kind: 19, Item: 2}},
	{`"action":"attack","max":1`, orderCall{Kind: 6, Subject: 91, Target: 201, Place: ai.ZoneRef{Space: ai.Coordinate{X: -3}}, Clear: true}},
	{`"action":"attack_space","max":1`, orderCall{Kind: 8, Subject: 91, Place: ai.ZoneRef{Space: ai.Coordinate{X: -4}}, Clear: true}},
	{`"action":"attack_zone","max":1`, orderCall{Kind: 9, Subject: 91, Place: ai.ZoneRef{Space: ai.Coordinate{X: -4}}, Clear: true}},
	{`"action":"garrison","max":1`, orderCall{Kind: 10, Subject: 73, Target: 800, Clear: true}}, {`"action":"ungarrison","max":1`, orderCall{Kind: 11, Subject: 101, Clear: true}},
	{`"action":"delete_unit","max":1`, orderCall{Kind: 12, Subject: 73, Clear: true}}, {`"action":"delete_building","max":1`, orderCall{Kind: 15, Subject: 800}},
	{`"action":"building_attack","max":1`, orderCall{Kind: 16, Subject: 800, Target: 201, Place: ai.ZoneRef{Space: ai.Coordinate{X: -3}}}},
}

func TestUnitBehaviorPayload(t *testing.T) {
	value, stance := false, ai.UnitStancePassive
	for _, filter := range []string{"", `,"unit_ids":[1]`, `,"unit_types":["economic"]`} {
		model := requireModel(t, `{"rules":[{"when":{"always":{}},"then":[{"action":"set_unit_behavior","stance":"passive","attack_back":false,"interrupt_current":false,"target_priority":[]`+filter+`}]}]}`)
		view := &fakeView{units: []ai.UnitView{{InternalID: 73, UnitID: 1, Kind: ai.UnitEconomic, Type: ai.UnitTypeEconomic, AttackBack: true}, {InternalID: 91, UnitID: 2, Kind: ai.UnitMilitary, Type: ai.UnitTypeInfantry, AttackBack: true, InterruptCurrent: true, TargetPriority: []ai.TargetCategory{ai.TargetCategoryMilitary}}}}
		want := ai.Decision{Orders: []ai.Order{}}
		if filter == "" {
			want.Behaviors = []ai.UnitBehavior{{Subjects: []uint64{91}, Stance: &stance, AttackBack: &value, InterruptCurrent: &value, TargetPriority: []ai.TargetCategory{}}}
		}
		if got := model.DecideOrders(view); !reflect.DeepEqual(got, want) || len(view.calls) != 0 {
			t.Fatalf("filter %s: %+v/%+v, want %+v", filter, got, view.calls, want)
		}
	}
}

func TestBuildingBehaviorPayload(t *testing.T) {
	value := false
	model := requireModel(t, `{"rules":[{"when":{"always":{}},"then":[{"action":"set_building_behavior","auto_attack":false,"interrupt_current":false,"target_priority":["military","building","economic"]}]}]}`)
	priority := []ai.TargetCategory{ai.TargetCategoryMilitary, ai.TargetCategoryBuilding, ai.TargetCategoryEconomic}
	view := &fakeView{buildings: []ai.BuildingView{{InternalID: 800, CanAttack: true, AutoAttack: true, InterruptCurrent: true}, {InternalID: 801, CanAttack: false, AutoAttack: true}, {InternalID: 802, CanAttack: true, TargetPriority: priority}}}
	want := ai.Decision{Orders: []ai.Order{}, BuildingBehaviors: []ai.BuildingBehavior{{Subjects: []uint64{800}, AutoAttack: &value, InterruptCurrent: &value, TargetPriority: priority}}}
	if got := model.DecideOrders(view); !reflect.DeepEqual(got, want) || len(view.calls) != 0 {
		t.Fatalf("got %+v/%+v, want %+v", got, view.calls, want)
	}
}

func TestUnitStopFiltersAndSharedOrders(t *testing.T) {
	for _, c := range []struct {
		options string
		ids     []uint64
	}{{``, []uint64{500, 501, 503}}, {`,"max":1`, []uint64{500, 501}}, {`,"unit_ids":[2]`, []uint64{500, 503}}} {
		model := requireModel(t, `{"rules":[{"when":{"always":{}},"then":[{"action":"unit_stop","orders":["gather"]`+c.options+`}]}]}`)
		view := &fakeView{units: []ai.UnitView{{InternalID: 91, UnitID: 1, ActiveOrders: []ai.ActiveUnitOrder{{ID: 500, Kind: ai.OrderKindGather}, {ID: 501, Kind: ai.OrderKindGather}, {ID: 502, Kind: ai.OrderKindMove}}}, {InternalID: 92, UnitID: 2, ActiveOrders: []ai.ActiveUnitOrder{{ID: 500, Kind: ai.OrderKindGather}, {ID: 503, Kind: ai.OrderKindGather}}}}}
		var calls []orderCall
		for _, id := range c.ids {
			calls = append(calls, orderCall{Kind: 20, Target: id})
		}
		assertCalls(t, model.DecideOrders(view), view, calls...)
	}
}

func TestRepeatedResearchReservation(t *testing.T) {
	model := requireModel(t, `{"rules":[{"when":{"always":{}},"then":[{"action":"research","tech_id":30},{"action":"research","tech_id":30}]}]}`)
	view := queueView()
	second := view.buildings[1]
	second.InternalID = 802
	view.buildings = append(view.buildings, second)
	view.resources[ai.ResourceWood] = 14
	assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: 14, Subject: 801, Item: 30})
}

func TestRepeatedMaintenanceReservation(t *testing.T) {
	for action, kind := range map[string]uint8{"repair": 4, "renew": 5} {
		model := requireModel(t, fmt.Sprintf(`{"rules":[{"when":{"always":{}},"then":[{"action":%q},{"action":%q}]}]}`, action, action))
		view := populatedView()
		view.stateful = true
		assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: kind, Subject: 73, Target: 800, Clear: true})
	}
}

func TestArmyLaunchBoundaries(t *testing.T) {
	for _, c := range []struct {
		distance, launch int
		attack           bool
	}{{2, 0, true}, {2, 1, true}, {2, 2, false}, {3, 0, false}, {3, 1, false}, {3, 2, false}} {
		model := requireModel(t, fmt.Sprintf(`{"rules":[{"when":{"always":{}},"then":[{"action":"army","launch":%d,"retreat":0,"defend_radius":2}]}]}`, c.launch))
		home, from, enemy := ai.Coordinate{}, ai.Coordinate{X: c.distance}, ai.Coordinate{X: 5}
		view := &fakeView{units: []ai.UnitView{{InternalID: 91, Kind: ai.UnitMilitary, Location: ai.ZoneRef{Space: from}}}, buildings: []ai.BuildingView{{InternalID: 800, BuildingID: 1}}, visibleEnemyUnits: []ai.UnitView{{InternalID: 201, Kind: ai.UnitMilitary, Location: ai.ZoneRef{Space: enemy}}}, spaceDistances: map[[2]ai.Coordinate]int{{from, from}: 0, {from, home}: c.distance, {home, enemy}: 5, {from, enemy}: 4}, locationDistances: map[[2]ai.ZoneRef]int{{{Space: from}, {Space: enemy}}: 8}}
		call := orderCall{Kind: 1, Subject: 91, Clear: true}
		if c.attack {
			call = orderCall{Kind: 6, Subject: 91, Target: 201, Place: ai.ZoneRef{Space: enemy}, Clear: true}
		}
		assertCalls(t, model.DecideOrders(view), view, call)
	}
}

func TestPickerIdentityTies(t *testing.T) {
	low, high := uint64(9007199254740993), uint64(9007199254740994)
	place := ai.ZoneRef{Space: ai.Coordinate{X: 5}}
	for _, spread := range []string{"", `,"spread":{"capacity":1,"overflow":"none"}`, `,"spread":{"capacity":0}`, `,"spread":{"capacity":0,"group_by":"space"}`} {
		for _, ids := range [][2]uint64{{high, low}, {low, high}} {
			model := requireModel(t, `{"rules":[{"when":{"always":{}},"then":[{"action":"attack","targets":"enemy_units","score":0,"min_score":0`+spread+`}]}]}`)
			view := &fakeView{units: []ai.UnitView{{InternalID: 91, Kind: ai.UnitMilitary}}, visibleEnemyUnits: []ai.UnitView{{InternalID: ids[0], Location: place}, {InternalID: ids[1], Location: place}}}
			assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: 6, Subject: 91, Target: low, Place: place, Clear: true})
		}
	}
}

func TestActionConstructors(t *testing.T) {
	for _, c := range constructorCases {
		t.Run(c.action, func(t *testing.T) {
			model, view := requireModel(t, `{"rules":[{"when":{"always":{}},"then":[{`+c.action+`}]}]}`), populatedView()
			assertCalls(t, model.DecideOrders(view), view, c.call)
		})
	}
}

func populatedView() *fakeView {
	garrison := uint64(800)
	return &fakeView{popLimit: 1, resources: map[ai.ResourceCategory]float64{ai.ResourceFood: 20, ai.ResourceWood: 20, ai.ResourceStone: 20, ai.ResourceGold: 15}, techResearched: map[uint32]bool{4: true}, unitCost: map[uint32]ai.Cost{2: {Food: 1, Wood: 2, Stone: 4, Gold: 2}},
		units:               []ai.UnitView{{InternalID: 73, UnitID: 1, Kind: ai.UnitEconomic, Type: ai.UnitTypeEconomic}, {InternalID: 91, UnitID: 2, Kind: ai.UnitMilitary, Type: ai.UnitTypeInfantry}, {InternalID: 101, UnitID: 1, Kind: ai.UnitEconomic, Type: ai.UnitTypeEconomic, GarrisonedIn: &garrison}},
		buildings:           []ai.BuildingView{{InternalID: 800, BuildingID: 1, Health: 5, MaxHealth: 10, Gatherable: true, RenewCost: ai.Cost{Wood: 2}, GarrisonCapacity: 1, CanAttack: true, Idle: true}},
		visibleEnemyUnits:   []ai.UnitView{{InternalID: 201, UnitID: 2, Kind: ai.UnitMilitary, Type: ai.UnitTypeInfantry, Location: ai.ZoneRef{Space: ai.Coordinate{X: -3}}}},
		knownEnemyBuildings: []ai.BuildingView{{InternalID: 301, BuildingID: 2, Location: ai.ZoneRef{Space: ai.Coordinate{X: -4}}}},
		knownResources:      []ai.ResourceView{{InternalID: 401, Category: ai.ResourceWood, AmountLeft: 31, Location: ai.ZoneRef{Space: ai.Coordinate{X: 2}}}},
		foundations:         []ai.FoundationView{{BuildingID: 2, Planned: true, Location: ai.ZoneRef{Space: ai.Coordinate{X: 3}}}},
	}
}

func queueView() *fakeView {
	return &fakeView{stateful: true, popLimit: 1, resources: map[ai.ResourceCategory]float64{ai.ResourceFood: 5, ai.ResourceWood: 7, ai.ResourceStone: 3},
		unitCost: map[uint32]ai.Cost{10: {Food: 5}, 11: {Food: 9}, 12: {Food: 5}}, techCost: map[uint32]ai.Cost{30: {Wood: 7}}, buildCost: map[uint32]ai.Cost{20: {Stone: 3}}, buildingAvailable: map[uint32]bool{20: true},
		units:     []ai.UnitView{{InternalID: 73, Kind: ai.UnitEconomic, Type: ai.UnitTypeEconomic, Builds: []ai.Producible{{Kind: ai.ProducibleBuilding, ID: 20}}}},
		buildings: []ai.BuildingView{{InternalID: 800, Idle: true, Producibles: []ai.Producible{{Kind: ai.ProducibleUnit, ID: 10, Cost: ai.Cost{Food: 5}}, {Kind: ai.ProducibleUnit, ID: 12, Cost: ai.Cost{Food: 5}}}}, {InternalID: 801, Idle: true, Producibles: []ai.Producible{{Kind: ai.ProducibleTech, ID: 30, Cost: ai.Cost{Wood: 7}}}}},
	}
}

func TestProductionResourceReservations(t *testing.T) {
	for _, c := range []struct {
		category ai.ResourceCategory
		cost     ai.Cost
		price    float64
	}{{ai.ResourceFood, ai.Cost{Food: 3}, 3}, {ai.ResourceWood, ai.Cost{Wood: 5}, 5}, {ai.ResourceStone, ai.Cost{Stone: 7}, 7}, {ai.ResourceGold, ai.Cost{Gold: 11}, 11}} {
		for _, balance := range []float64{c.price - 0.25, c.price} {
			model := requireModel(t, `{"rules":[{"when":{"always":{}},"then":[{"action":"create","unit_id":10,"queue":2},{"action":"create","unit_id":10,"queue":2}]}]}`)
			view := &fakeView{stateful: true, popLimit: 2, resources: map[ai.ResourceCategory]float64{c.category: balance}, buildings: []ai.BuildingView{{InternalID: 800, Idle: true, Producibles: []ai.Producible{{Kind: ai.ProducibleUnit, ID: 10, Cost: c.cost}}}}}
			var calls []orderCall
			if balance == c.price {
				calls = append(calls, orderCall{Kind: 13, Subject: 800, Item: 10})
			}
			assertCalls(t, model.DecideOrders(view), view, calls...)
		}
	}
}

func TestActionsWithEmptyView(t *testing.T) {
	for _, action := range actionFixtures {
		t.Run(action, func(t *testing.T) {
			model := requireModel(t, fmt.Sprintf(`{"rules":[{"when":{"always":null},"then":[%s]}]}`, action))
			view := &fakeView{}
			assertCalls(t, model.DecideOrders(view), view)
		})
	}
}

func assertCalls(t *testing.T, decision ai.Decision, view *fakeView, calls ...orderCall) {
	t.Helper()
	orders := make([]ai.Order, 0, len(calls))
	for index, call := range calls {
		orders = append(orders, ai.Order{Subjects: []uint64{call.Subject}, OrderType: call.Kind, TargetID: int64(index + 1), ClearPreviousOrders: call.Clear})
	}
	want := ai.Decision{Orders: orders}
	if !reflect.DeepEqual(decision, want) || !reflect.DeepEqual(view.calls, calls) {
		t.Fatalf("decision %+v, calls %+v; want %+v, calls %+v", decision, view.calls, want, calls)
	}
}

func parseAndRun(t *testing.T, actionMap map[string]any, view ai.View) ai.Decision {
	rng := rand.New(rand.NewSource(1))
	raw := map[string]any{
		"rules": []any{
			map[string]any{
				"when": map[string]any{"always": map[string]any{}},
				"then": []any{actionMap},
			},
		},
	}
	model := ai.ParseModelWithDiagnostics(raw, rng, io.Discard)
	if _, ok := model.(*ai.RulesModel); !ok {
		t.Fatalf("expected RulesModel, got %T", model)
	}
	return model.DecideOrders(view)
}

func TestGatherActionOrders(t *testing.T) {
	view := &fakeView{
		units: []ai.UnitView{
			{InternalID: 1, Type: ai.UnitTypeEconomic, Kind: ai.UnitEconomic},
		},
		knownResources: []ai.ResourceView{
			{Category: ai.ResourceWood, AmountLeft: 100},
		},
	}

	action := map[string]any{
		"action":   "gather",
		"category": "wood",
	}

	decision := parseAndRun(t, action, view)

	if len(decision.Orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(decision.Orders))
	}
}

func TestCreateActionOrders(t *testing.T) {
	view := &fakeView{
		buildings: []ai.BuildingView{
			{InternalID: 1, Idle: true, Producibles: []ai.Producible{{Kind: ai.ProducibleUnit, ID: 10}}},
		},
		popLimit:   10,
		popCurrent: 0,
		resources: map[ai.ResourceCategory]float64{
			ai.ResourceFood: 100, ai.ResourceWood: 100, ai.ResourceStone: 100, ai.ResourceGold: 100,
		},
	}

	action := map[string]any{
		"action":  "create",
		"unit_id": 10.0,
	}

	decision := parseAndRun(t, action, view)

	if len(decision.Orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(decision.Orders))
	}
}

func TestResearchActionOrders(t *testing.T) {
	view := &fakeView{
		buildings: []ai.BuildingView{
			{InternalID: 1, Idle: true, Producibles: []ai.Producible{{Kind: ai.ProducibleTech, ID: 10}}},
		},
		resources: map[ai.ResourceCategory]float64{
			ai.ResourceFood: 100, ai.ResourceWood: 100, ai.ResourceStone: 100, ai.ResourceGold: 100,
		},
	}

	action := map[string]any{
		"action":  "research",
		"tech_id": 10.0,
	}

	decision := parseAndRun(t, action, view)

	if len(decision.Orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(decision.Orders))
	}
}

func TestBuildActionOrders(t *testing.T) {
	view := &fakeView{
		units: []ai.UnitView{
			{
				InternalID: 1,
				Type:       ai.UnitTypeEconomic,
				Kind:       ai.UnitEconomic,
				Builds: []ai.Producible{
					{Kind: ai.ProducibleBuilding, ID: 20},
				},
			},
		},
		resources: map[ai.ResourceCategory]float64{
			ai.ResourceFood: 100, ai.ResourceWood: 100, ai.ResourceStone: 100, ai.ResourceGold: 100,
		},
		buildingAvailable: map[uint32]bool{20: true},
	}

	action := map[string]any{
		"action":      "build",
		"building_id": 20.0,
	}

	decision := parseAndRun(t, action, view)

	if len(decision.Orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(decision.Orders))
	}
}

func TestAttackActionOrders(t *testing.T) {
	view := &fakeView{
		units: []ai.UnitView{
			{
				InternalID: 1,
				Type:       ai.UnitTypeInfantry,
				Kind:       ai.UnitMilitary,
			},
		},
		visibleEnemyUnits: []ai.UnitView{
			{InternalID: 2, Kind: ai.UnitMilitary},
		},
	}

	action := map[string]any{
		"action":  "attack",
		"targets": "enemy_units",
	}

	decision := parseAndRun(t, action, view)

	if len(decision.Orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(decision.Orders))
	}
}

func TestGarrisonActionOrders(t *testing.T) {
	view := &fakeView{
		units: []ai.UnitView{
			{InternalID: 1, Kind: ai.UnitEconomic}, // No garrison target logic for tests, just dummy match
		},
		buildings: []ai.BuildingView{
			{InternalID: 2, GarrisonCapacity: 10, GarrisonCount: 0},
		},
	}

	action := map[string]any{
		"action": "garrison",
	}

	decision := parseAndRun(t, action, view)

	if len(decision.Orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(decision.Orders))
	}
}

func TestUngarrisonActionOrders(t *testing.T) {
	garrisonID := uint64(2)
	view := &fakeView{
		units: []ai.UnitView{
			{InternalID: 1, Kind: ai.UnitEconomic, GarrisonedIn: &garrisonID},
		},
	}

	action := map[string]any{
		"action": "ungarrison",
	}

	decision := parseAndRun(t, action, view)

	if len(decision.Orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(decision.Orders))
	}
}
