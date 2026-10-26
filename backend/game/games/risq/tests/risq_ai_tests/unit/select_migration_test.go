package unit

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	. "github.com/dgray001/gray_online/game/games/risq/ai"
)

func TestCurrentOrderCoordinatesIncludeEntityTargets(t *testing.T) {
	loc := ZoneRef{Space: Coordinate{X: 7, Y: 3}, Zone: Coordinate{X: 1, Y: -1}}
	for name, order := range map[string]*CurrentOrder{
		"zone": {TargetZone: &loc}, "space": {TargetSpace: &loc.Space},
		"resource": {TargetResource: &ResourceView{Location: loc}},
		"unit":     {TargetUnit: &UnitView{Location: loc}},
		"building": {TargetBuilding: &BuildingView{Location: loc}},
	} {
		view := tieView()
		view.units[0].CurrentOrder = order
		config := rule(`{"action":"set_var","name":"location","value":{"sum":{"kind":"units","of":"100 * var(item.order_space_x) + var(item.order_space_y)"}}},
			{"action":"set_var","name":"zone","value":{"sum":{"kind":"units","of":"10 * var(item.order_zone_x) + var(item.order_zone_y)"}}}`)
		got := decide(t, config, view, "location", "zone")
		wantZone := 9.0
		if name == "space" {
			wantZone = 0
		}
		if got["location"] != 703 || got["zone"] != wantZone {
			t.Errorf("%s: target location = %v zone = %v, want 703 and %v", name, got["location"], got["zone"], wantZone)
		}
	}
}

func TestSelectUnbucketedUnits(t *testing.T) {
	view := productionView()
	view.visibleEnemyUnits = nil
	view.units = []UnitView{{InternalID: 91}, {InternalID: 92}, {InternalID: 93}}
	config := rule(`{"action":"select","pick":{"kind":"units","where":{"bucket":"none"}},"into":{"set":"initial","var":"initial_count"}},
		{"action":"select","pick":{"kind":"units","where":{"in":{"uid":[91]}}},"into":{"bucket":"a","mode":"add"}},
		{"action":"select","pick":{"kind":"units","where":{"in":{"uid":[91,92]}}},"into":{"bucket":"b","mode":"add"}},
		{"action":"select","pick":{"kind":"units","where":{"bucket":"none"}},"into":{"set":"free","var":"free_count"}},
		{"action":"select","pick":{"kind":"units","where":{"bucket":"a"}},"into":{"set":"named","var":"named_count"}},
		{"action":"select","pick":{"kind":"units","where":{"in":{"uid":[91]}}},"into":{"bucket":"all","mode":"remove"}},
		{"action":"select","pick":{"kind":"units","where":{"bucket":"none"}},"into":{"set":"freed","var":"freed_count"}}`)
	got := decide(t, config, view, "initial_count", "free_count", "named_count", "freed_count")
	for name, want := range map[string]float64{"initial_count": 3, "free_count": 1, "named_count": 1, "freed_count": 2} {
		if got[name] != want {
			t.Errorf("%s = %v, want %v", name, got[name], want)
		}
	}
}

func TestSelectSetBehaviorUsesSelectedActorTypes(t *testing.T) {
	view := productionView()
	model := actionsModel(t, `{"action":"select","actors":{"kind":"units"},"issue":[{"set_behavior":{"stance":"passive","attack_back":false,"interrupt_current":false,"target_priority":[]}}]},
		{"action":"select","actors":{"kind":"buildings","where":{"in":{"uid":[1000000800]}}},"issue":[{"set_behavior":{"auto_attack":false,"target_priority":["military"]}}]}`)
	decision := model.DecideOrders(view)
	value, stance := false, UnitStancePassive
	units := []UnitBehavior{{Subjects: []uint64{91}, Stance: &stance, AttackBack: &value, InterruptCurrent: &value, TargetPriority: []TargetCategory{}}}
	buildings := []BuildingBehavior{{Subjects: []uint64{800}, AutoAttack: &value, TargetPriority: []TargetCategory{TargetCategoryMilitary}}}
	if !reflect.DeepEqual(decision.Behaviors, units) || !reflect.DeepEqual(decision.BuildingBehaviors, buildings) || len(decision.Orders) != 0 {
		t.Fatalf("unexpected control decision: %+v", decision)
	}
}

func TestSelectBehaviorKeepsLaterOverrides(t *testing.T) {
	model := actionsModel(t, `{"action":"select","actors":{"kind":"units"},"issue":[{"set_behavior":{"stance":"aggressive"}},{"set_behavior":{"stance":"passive"}}]}`)
	decision := model.DecideOrders(productionView())
	if len(decision.Behaviors) != 2 || *decision.Behaviors[1].Stance != UnitStancePassive {
		t.Fatalf("later behavior override lost: %+v", decision.Behaviors)
	}
}

func TestSelectStopFiltersAndDeduplicatesSharedUnitOrders(t *testing.T) {
	view := productionView()
	view.allowed = nil
	view.units = []UnitView{
		{InternalID: 91, ActiveOrders: []ActiveUnitOrder{{ID: 500, Kind: OrderKindGather}, {ID: 501, Kind: OrderKindMove}}},
		{InternalID: 92, ActiveOrders: []ActiveUnitOrder{{ID: 500, Kind: OrderKindGather}, {ID: 502, Kind: OrderKindGather}}},
	}
	stop := `{"action":"select","actors":{"kind":"units"},"issue":[{"stop":{"orders":["gather"]}}]}`
	model := actionsModel(t, stop+`,`+stop)
	assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: 20, Target: 500}, orderCall{Kind: 20, Target: 502})
	if len(view.units[0].ActiveOrders) != 1 || view.units[0].ActiveOrders[0].ID != 501 || len(view.units[1].ActiveOrders) != 0 {
		t.Fatalf("unexpected remaining orders: %+v", view.units)
	}
}

func TestNearSupportsUnitTypesAndBuildingIDs(t *testing.T) {
	view := selectTestView{loopTestView: loopTestView{
		spaces:      lineOfSpaces(0, 0, 0),
		enemy_units: []UnitView{{InternalID: 1, PlayerID: 1, Type: UnitTypeArcher, Location: at(2)}, {InternalID: 2, PlayerID: 1, Type: UnitTypeInfantry, Location: at(3)}},
		buildings:   []BuildingView{{InternalID: 3, BuildingID: 9, Location: at(1)}, {InternalID: 4, BuildingID: 10, Location: at(3)}},
	}}
	for _, origin := range []string{
		`{"kind":"units","where":{"owner":"enemy","unit_types":["archer"]}}`,
		`{"kind":"units","where":{"owner":"enemy","in":{"unit_type":["archer"]}}}`,
		`{"kind":"buildings","where":{"building_ids":[9]}}`,
	} {
		config := rule(`{"action":"select","pick":{"kind":"spaces","where":{"near":{"distance":0,"of":` + origin + `}}},"into":{"set":"chosen","var":"n"}}`)
		if got := decide(t, config, view, "n")["n"]; got != 1 {
			t.Errorf("%s: selected %v spaces, want one", origin, got)
		}
	}
}

func TestSelectFailedAssignmentExpressionsEmitNothing(t *testing.T) {
	worker := villager(1, Coordinate{})
	view := selectTestView{loopTestView: loopTestView{units: []UnitView{worker}, spaces: lineOfSpaces(0, 0)}, idle: []UnitView{worker}}
	for _, conf := range []string{
		`{"action":"select","actors":{"kind":"units","rank":"1 / 0"},"pick":{"kind":"spaces"},"issue":[{"move":{}}]}`,
		`{"action":"select","pick":{"kind":"spaces","capacity":"1 / 0"},"issue":[{"move":{}}]}`,
		`{"action":"select","actors":{"kind":"units"},"pick":{"kind":"spaces"},"match":{"weight":"1 / 0"},"issue":[{"move":{}}]}`,
		`{"action":"select","actors":{"kind":"units"},"pick":{"kind":"spaces"},"match":{"score":"1 / 0"},"issue":[{"move":{}}]}`,
	} {
		config := rule(conf + `,{"action":"select","pick":{"kind":"spaces","where":{"in":{"x":[2]}}},"issue":[{"move":{}}]}`)
		if orders := selectOrders(t, config, view); len(orders) != 1 || orders[0].TargetID != spaceCode(at(2).Space) {
			t.Errorf("failed expr in %s: got %+v, want only the subsequent valid action", conf, orders)
		}
	}
}

func TestSelectHireCountsUseEachZonesRemainingBalance(t *testing.T) {
	config := rule(`{"action":"select","pick":{"kind":"zones","rank":"var(target.space_gold_income)","limit":3,
		"let":{"requested":"var(target.space_gold_income)"}},"issue":[{"hire":{"unit_id":101,"count":"min(var(target.requested), var(item.can_afford))"}}]},
		{"action":"set_var","name":"left","value":"gameVar(gold) + 100 * gameVar(population)"}`)
	model := parseSelectModel(t, config)
	orders := model.DecideOrders(decisionView{hireView()}).Orders
	if left, _ := model.Var("left"); len(orders) != 16 || left != 1604 {
		t.Fatalf("orders = %+v and left = %v, want 16 hires, 4 gold remaining and 16 pending population", orders, left)
	}
	if counts := hireCounts(orders); counts[200] != 9 || counts[300] != 7 || len(counts) != 2 {
		t.Errorf("hires per zone = %v, want 9 at the first zone and 7 at the next", counts)
	}
}

func TestSelectHireSkipsOnlyZonesWithFailedOrNonpositiveCounts(t *testing.T) {
	config := rule(`{"action":"select","pick":{"kind":"zones","rank":"var(target.x)"},
		"issue":[{"hire":{"unit_id":101,"count":"1 / (var(target.x) - 2)"}}]},
		{"action":"set_var","name":"left","value":"gameVar(gold) + 100 * gameVar(population)"}`)
	model := parseSelectModel(t, config)
	orders := model.DecideOrders(decisionView{hireView()}).Orders
	if left, _ := model.Var("left"); len(orders) != 2 || left != 288 {
		t.Fatalf("orders = %+v and left = %v, want two hires, 88 gold remaining and two pending population", orders, left)
	}
	if counts := hireCounts(orders); counts[400] != 1 || counts[300] != 1 || len(counts) != 2 {
		t.Errorf("hires per zone = %v, want only zones 4 and 3", counts)
	}
}

func TestSelectFailedHireCountEmitsNothingAndReservesNothing(t *testing.T) {
	for _, count := range []string{`"1 / 0"`, `"var(item.can_afford) / 0"`} {
		config := rule(`{"action":"select","pick":{"kind":"zones"},"issue":[{"hire":{"unit_id":101,"count":` + count + `}}]},
			{"action":"set_var","name":"bank","value":"gameVar(gold)"},
			{"action":"set_var","name":"planned_population","value":"gameVar(population)"}`)
		vars := decide(t, config, hireView(), "bank", "planned_population")
		if vars["bank"] != 100 || vars["planned_population"] != 0 {
			t.Errorf("count %s: failed hire count left reservations or orders: %v", count, vars)
		}
	}
}

func TestSelectAssignmentSkipsActorsBlockedByValidation(t *testing.T) {
	workers := []UnitView{{InternalID: 1, Type: UnitTypeInfantry}, villager(2, Coordinate{})}
	view := selectTestView{loopTestView: loopTestView{units: workers, buildings: []BuildingView{{InternalID: 1}}}, idle: workers}
	for _, verb := range []string{"gather", "repair", "renew"} {
		config := rule(fmt.Sprintf(`{"action":"select","pick":{"kind":"buildings","capacity":1,"where":{"in":{"uid":[1000000001]}}},"actors":{"kind":"units","rank":"0 - var(actor.uid)"},"issue":[{%q:{}}]}`, verb))
		orders := selectOrders(t, config, view)
		if len(orders) != 1 || !slices.Equal(orders[0].Subjects, []uint64{2}) || !orders[0].ClearPreviousOrders {
			t.Errorf("%s: got %+v, want one clearing order for villager 2", verb, orders)
		}
	}
}

func TestSelectNestedSetKindMismatchStopsAction(t *testing.T) {
	worker := villager(1, at(1).Space)
	view := selectTestView{loopTestView: loopTestView{units: []UnitView{worker}, spaces: lineOfSpaces(0)}, idle: []UnitView{worker}}
	for _, selectors := range []string{
		`"pick":{"kind":"spaces","rank":{"count":{"kind":"buildings","where":{"not_in_set":"chosen"}}}}`,
		`"pick":{"kind":"spaces"},"actors":{"kind":"units","rank":{"count":{"kind":"buildings","where":{"not_in_set":"chosen"}}}}`,
	} {
		config := rule(`{"action":"select","pick":{"kind":"units"},"into":{"set":"chosen"}},
			{"action":"select",` + selectors + `,"issue":[{"move":{}}]}`)
		if orders := selectOrders(t, config, view); len(orders) != 0 {
			t.Errorf("%s: emitted %+v", selectors, orders)
		}
	}
}

func foundationView() selectTestView {
	near, far, started := villager(1, Coordinate{}), villager(2, Coordinate{X: 5}), villager(3, Coordinate{X: 9})
	return selectTestView{
		loopTestView: loopTestView{units: []UnitView{near, far, started}},
		idle:         []UnitView{near, far},
		foundations: []FoundationView{
			{BuildingID: 2, Location: ZoneRef{Space: Coordinate{X: 1}}, Planned: true},
			{BuildingID: 2, Location: ZoneRef{Space: Coordinate{X: 6}}, Planned: true, Builders: 1},
			{BuildingID: 3, Location: ZoneRef{Space: Coordinate{X: 7}}},
		},
		turn: 1,
	}
}

func TestSelectBuildersAssistFoundationsWithoutBuilders(t *testing.T) {
	config := rule(`{"action":"select","pick":{"kind":"foundations","capacity":1,"where":{"building_ids":[2],"if":"1 - var(target.builders)"}},
		"actors":{"kind":"units","where":{"if":"var(actor.idle)"},"rank":"0 - var(actor.x)"},"match":{"score":"0 - distance(actor, target)"},
		"issue":[{"build":{}}]}`)
	orders := selectOrders(t, config, foundationView())
	if len(orders) != 1 || orders[0].OrderType != kindBuild || orders[0].TargetID != 100 || orders[0].Subjects[0] != 1 {
		t.Fatalf("orders = %+v, want only the unstaffed building-2 foundation at x=1 taken by the nearest idle builder", orders)
	}
}

func TestSelectCancelsPlannedFoundationsAndRefundsThem(t *testing.T) {
	config := rule(`{"action":"select","pick":{"kind":"foundations","where":{"if":"1 - var(target.builders)"}},"issue":[{"cancel_foundation":{}}]}`)
	var cancelled []int64
	for _, o := range selectOrders(t, config, foundationView()) {
		if o.OrderType != kindCancelFoundation {
			t.Fatalf("unexpected order %+v", o)
		}
		cancelled = append(cancelled, o.TargetID)
	}
	if !slices.Equal(cancelled, []int64{100}) {
		t.Fatalf("cancelled foundations = %v, want only the planned, unstaffed one at x=1 (the other has a builder, and the third is already started)", cancelled)
	}
}

func TestSelectDoesNotCancelFoundationsPlacedThisTurn(t *testing.T) {
	view := foundationView()
	view.foundations = append(view.foundations, FoundationView{BuildingID: 2, Location: ZoneRef{Space: Coordinate{X: 3}}, Planned: true, OrderedThisTurn: true})
	config := rule(`{"action":"select","pick":{"kind":"foundations","where":{"if":"1 - var(target.builders)"}},"issue":[{"cancel_foundation":{}}]}`)
	orders := selectOrders(t, config, view)
	if len(orders) != 1 || orders[0].TargetID != 100 {
		t.Fatalf("orders = %+v, want only the server-known foundation at x=1 cancelled, not the one placed this turn", orders)
	}
}

func TestForEachFoundationFields(t *testing.T) {
	config := rule(`{"action":"for_each","source":"foundations","as":"f","do":[{"action":"set_var","name":"sum","value":"var(sum) + var(f.building_id) * 100 + var(f.is_planned) * 10 + var(f.builders)","persist":true}]}`)
	if got := decide(t, config, foundationView(), "sum")["sum"]; got != 200+10+200+11+300 {
		t.Errorf("foundation field sum = %v, want %d", got, 200+10+200+11+300)
	}
}

func TestSelectActorsStayInsideTheBucketScope(t *testing.T) {
	units := []UnitView{villager(1, Coordinate{}), villager(2, Coordinate{}), villager(3, Coordinate{})}
	view := selectTestView{
		loopTestView: loopTestView{units: units, spaces: lineOfSpaces(0, 0)},
		idle:         units, turn: 1,
	}
	config := `{"rules": [{"when": {"always": {}}, "then": [
		{"action":"select","pick":{"kind":"units","limit":1},"into":{"bucket":"squad"}},
		{"action":"select","actors":{"kind":"units","where":{"bucket":"squad"}},"pick":{"kind":"spaces","limit":1},"issue":[{"move":{}}]}]}]}`
	orders := selectOrders(t, config, view)
	if len(orders) != 1 {
		t.Fatalf("orders = %+v, want only the single bucket member to be ordered, not every idle unit", orders)
	}
}

func TestGarrisonCapacityIsPlainJSONOverBuildingFields(t *testing.T) {
	units := []UnitView{villager(1, Coordinate{}), villager(2, Coordinate{}), villager(3, Coordinate{}), villager(4, Coordinate{})}
	view := selectTestView{
		loopTestView: loopTestView{
			units: units,
			buildings: []BuildingView{
				{InternalID: 10, BuildingID: 5, GarrisonCount: 1, GarrisonCapacity: 3},
				{InternalID: 11, BuildingID: 5, GarrisonCount: 2, GarrisonCapacity: 2},
			},
		},
		idle: units, turn: 1,
	}
	config := rule(`{"action":"select","pick":{"kind":"buildings","where":{"owner":"own"},"capacity":"var(target.garrison_capacity) - var(target.garrison_count)"},"issue":[{"garrison":{}}]}`)
	orders := selectOrders(t, config, view)
	if len(orders) != 2 {
		t.Fatalf("orders = %+v, want exactly the two free slots of building 10 filled", orders)
	}
	for _, o := range orders {
		if o.OrderType != kindGarrison || o.TargetID != 10 {
			t.Errorf("order %+v, want a garrison into building 10", o)
		}
	}
}

func TestVerbsOnlyGuardWhatEngineValidationWouldRejectWholesale(t *testing.T) {
	fighter := UnitView{InternalID: 1, Type: UnitTypeInfantry, Kind: UnitMilitary, CanAttack: true}
	villager := villager(2, Coordinate{})
	unfinished := BuildingView{InternalID: 30, PlayerID: 1, UnderConstruction: true, Location: at(3)}
	view := selectTestView{
		loopTestView: loopTestView{
			units:           []UnitView{fighter, villager},
			buildings:       []BuildingView{{InternalID: 40, BuildingID: 7, UnderConstruction: true}, {InternalID: 41, BuildingID: 8, Gatherable: true}},
			enemy_buildings: []BuildingView{unfinished},
			enemy_units:     []UnitView{{InternalID: 60, PlayerID: 1, Location: at(3)}},
		},
		idle: []UnitView{fighter, villager}, turn: 1,
	}
	issued := func(pick string, verb string) map[uint64]bool {
		config := rule(`{"action":"select","pick":` + pick + `,"issue":[{"` + verb + `":{}}]}`)
		who := map[uint64]bool{}
		for _, o := range selectOrders(t, config, view) {
			who[o.Subjects[0]] = true
		}
		return who
	}
	if got := issued(`{"kind":"buildings","where":{"owner":"enemy"}}`, "attack"); !got[1] || got[2] {
		t.Errorf("units attacking an enemy building under construction: %v, want only the fighter, and the villager (which cannot attack) skipped", got)
	}
	if got := issued(`{"kind":"buildings","where":{"owner":"own","in":{"building_id":[7]}}}`, "repair"); !got[2] || got[1] {
		t.Errorf("repair of a building under construction issued for %v, want only the villager (validation does not reject it)", got)
	}
	if got := issued(`{"kind":"buildings","where":{"owner":"own","in":{"building_id":[8]}}}`, "renew"); !got[2] || got[1] {
		t.Errorf("renew of a gatherable building issued for %v, want only the villager", got)
	}
	if got := issued(`{"kind":"buildings","where":{"owner":"own","in":{"building_id":[7]}}}`, "renew"); !got[2] || got[1] {
		t.Errorf("renew of a non-gatherable building issued for %v, want only the villager (validation does not reject it)", got)
	}
}

func TestGatheringWorksOnGatherableBuildingsLikeFarms(t *testing.T) {
	worker := villager(1, Coordinate{})
	farm := BuildingView{InternalID: 80, BuildingID: 3, Gatherable: true, ResourcesLeft: 50, Location: at(2), ResourceCategory: ResourceFood}
	spent := BuildingView{InternalID: 81, BuildingID: 3, Gatherable: true, ResourcesLeft: 0, Location: at(4)}
	house := BuildingView{InternalID: 82, BuildingID: 2, Location: at(5)}
	tree := ResourceView{InternalID: 90, Category: ResourceWood, Location: at(3)}
	view := selectTestView{
		loopTestView: loopTestView{
			units:     []UnitView{worker},
			buildings: []BuildingView{farm, spent, house},
			spaces:    lineOfSpaces(0, 0, 0, 0, 0),
			zones:     []ZoneInfo{{Location: house.Location}},
			resources: map[ResourceCategory][]ResourceView{ResourceWood: {tree}},
		},
		idle: []UnitView{worker}, turn: 1,
	}
	gathered := func(pick string) []int64 {
		ids := []int64{}
		for _, o := range selectOrders(t, rule(`{"action":"select","pick":`+pick+`,"issue":[{"gather":{}}]}`), view) {
			if o.OrderType != kindGather {
				t.Fatalf("unexpected order %+v", o)
			}
			ids = append(ids, o.TargetID)
		}
		return ids
	}
	if got := gathered(`{"kind":"buildings","where":{"owner":"own","in":{"building_id":[3]},"if":"var(target.resources_left)"}}`); len(got) != 1 || got[0] != 80 {
		t.Errorf("gather from the farm that has food left = %v, want just building 80", got)
	}
	if got := gathered(`{"kind":"buildings","where":{"owner":"own","in":{"building_id":[2]}}}`); len(got) != 1 || got[0] != 82 {
		t.Errorf("gather from a house = %v, want it issued, since validation only needs a known building in the zone", got)
	}
	for _, kind := range []string{"spaces", "zones"} {
		if got := gathered(fmt.Sprintf(`{"kind":%q,"where":{"in":{"x":[5]}}}`, kind)); len(got) != 1 || got[0] != 82 {
			t.Errorf("gather from the house's %s = %v, want the known building regardless of gatherability", kind, got)
		}
	}
	if got := gathered(`{"kind":"spaces","where":{"in":{"x":[2]}},"limit":1}`); len(got) != 1 || got[0] != 80 {
		t.Errorf("gather from the space holding the farm = %v, want the farm", got)
	}
	if got := gathered(`{"kind":"spaces","where":{"in":{"x":[4]}},"limit":1}`); len(got) != 1 || got[0] != 81 {
		t.Errorf("gather from a space whose farm is spent = %v, want the farm, since depletion is not a validation error", got)
	}
	if got := gathered(`{"kind":"spaces","where":{"in":{"x":[3]}},"limit":1}`); len(got) != 1 || got[0] != 90 {
		t.Errorf("gather from the space holding a tree = %v, want the tree", got)
	}
}

func TestBuildNeedsAnExplicitValidZone(t *testing.T) {
	worker := villager(1, Coordinate{})
	open, blocked := ZoneRef{Space: Coordinate{X: 1}}, ZoneRef{Space: Coordinate{X: 2}}
	view := selectTestView{
		loopTestView: loopTestView{units: []UnitView{worker}, spaces: lineOfSpaces(0, 0, 0), zones: []ZoneInfo{{Location: open}, {Location: blocked}}},
		idle:         []UnitView{worker}, turn: 1, blocked_zones: []ZoneRef{blocked},
	}
	built := func(kind string) []Order {
		return selectOrders(t, rule(`{"action":"select","pick":{"kind":"`+kind+`"},"issue":[{"build":{"building_id":2}}]}`), view)
	}
	if got := built("zones"); len(got) != 1 || got[0].TargetID != spaceCode(open.Space) {
		t.Errorf("build over zones = %+v, want one order for the open zone and none for the blocked one", got)
	}
	if got := built("spaces"); len(got) != 0 {
		t.Errorf("build over spaces = %+v, want none, since a site must be an explicit zone", got)
	}
}

func TestRenewChargesItsCostOncePerBuildingUnlessAlreadyRenewing(t *testing.T) {
	workers := []UnitView{villager(1, Coordinate{}), villager(2, Coordinate{})}
	farm := BuildingView{InternalID: 80, BuildingID: 3, Gatherable: true, RenewCost: Cost{Wood: 10}}
	renewing := BuildingView{InternalID: 81, BuildingID: 3, Gatherable: true, Renewing: true, RenewCost: Cost{Wood: 10}}
	view := selectTestView{loopTestView: loopTestView{units: workers, buildings: []BuildingView{farm, renewing}}, idle: workers, turn: 1}
	renew := func(id int) string {
		return `{"action":"select","pick":{"kind":"buildings","where":{"owner":"own","in":{"uid":[` + fmt.Sprint(1_000_000_000+id) + `]}}},"issue":[{"renew":{}}]}`
	}
	probe := `{"action":"set_var","name":"wood_left","value":"gameVar(wood)"}`
	config := `{"rules":[{"when":{"always":{}},"then":[` + renew(80) + `,` + renew(81) + `,` + probe + `]}]}`
	model := parseSelectModel(t, config)
	model.DecideOrders(decisionView{view})
	if got, _ := model.Var("wood_left"); got != 90 {
		t.Errorf("wood after renewing two buildings with two villagers = %v, want 90 (one charge for the building that was not renewing)", got)
	}
}

func TestBuildingActorsAreAlwaysTheirOwn(t *testing.T) {
	view := selectTestView{
		loopTestView: loopTestView{
			buildings:       []BuildingView{{InternalID: 70, BuildingID: 9, CanAttack: true}},
			enemy_buildings: []BuildingView{{InternalID: 71, PlayerID: 1, BuildingID: 9, CanAttack: true}},
			enemy_units:     []UnitView{{InternalID: 60, PlayerID: 1, Location: at(1)}},
		},
		turn: 1,
	}
	config := rule(`{"action":"select","pick":{"kind":"units","where":{"owner":"enemy"}},
		"actors":{"kind":"buildings","where":{"owner":"any","in":{"building_id":[9]}}},"issue":[{"attack":{}}]}`)
	orders := selectOrders(t, config, view)
	if len(orders) != 1 || orders[0].Subjects[0] != 70 {
		t.Fatalf("orders = %+v, want only the own building 70, since an enemy subject rejects the whole batch", orders)
	}
}

func TestActorLimitAppliesAfterTheOwnershipFilter(t *testing.T) {
	view := selectTestView{
		loopTestView: loopTestView{
			buildings:       []BuildingView{{InternalID: 70, BuildingID: 9, CanAttack: true}},
			enemy_buildings: []BuildingView{{InternalID: 71, PlayerID: 1, BuildingID: 9, CanAttack: true}},
			enemy_units:     []UnitView{{InternalID: 60, PlayerID: 1, Location: at(1)}},
		},
		turn: 1,
	}
	config := rule(`{"action":"select","pick":{"kind":"units","where":{"owner":"enemy"}},
		"actors":{"kind":"buildings","where":{"owner":"any","in":{"building_id":[9]}},"rank":"var(actor.player)","limit":1},"issue":[{"attack":{}}]}`)
	orders := selectOrders(t, config, view)
	if len(orders) != 1 || orders[0].Subjects[0] != 70 {
		t.Fatalf("orders = %+v, want the own building 70: the enemy building ranks first but must not use up the limit", orders)
	}
}

func TestActorLimitAppliesAfterTheBucketScope(t *testing.T) {
	units := []UnitView{villager(1, Coordinate{}), villager(2, Coordinate{}), villager(3, Coordinate{})}
	view := selectTestView{loopTestView: loopTestView{units: units, spaces: lineOfSpaces(0, 0)}, idle: units, turn: 1}
	ordered := map[uint64]bool{}
	for _, rank := range []string{"var(actor.uid)", "0 - var(actor.uid)"} {
		config := `{"rules": [{"when": {"always": {}}, "then": [
			{"action":"select","pick":{"kind":"units","limit":1},"into":{"bucket":"squad"}},
			{"action":"select","pick":{"kind":"spaces","limit":1},
				"actors":{"kind":"units","where":{"bucket":"squad"},"rank":"` + rank + `","limit":1},"issue":[{"move":{}}]}]}]}`
		orders := selectOrders(t, config, view)
		if len(orders) != 1 {
			t.Errorf("rank %q: orders = %+v, want the one bucket member ordered, whichever unit ranks first", rank, orders)
			continue
		}
		ordered[orders[0].Subjects[0]] = true
	}
	if len(ordered) > 1 {
		t.Errorf("ordered units %v, want the same bucket member under both rankings", ordered)
	}
}

func TestAnUnfinishedBuildingCanStillBeOrderedToAttack(t *testing.T) {
	tower := BuildingView{InternalID: 70, BuildingID: 9, CanAttack: true, UnderConstruction: true}
	view := selectTestView{
		loopTestView: loopTestView{buildings: []BuildingView{tower}, enemy_units: []UnitView{{InternalID: 60, PlayerID: 1, Location: at(1)}}},
		turn:         1,
	}
	config := rule(`{"action":"select","pick":{"kind":"units","where":{"owner":"enemy"}},
		"actors":{"kind":"buildings","where":{"in":{"building_id":[9]}}},"issue":[{"attack":{}}]}`)
	orders := selectOrders(t, config, view)
	if len(orders) != 1 || orders[0].OrderType != kindBuildingAttack {
		t.Fatalf("orders = %+v, want the building ordered to attack; whether it can is the engine's call at receipt", orders)
	}
}
