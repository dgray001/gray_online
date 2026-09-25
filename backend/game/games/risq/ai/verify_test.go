package ai

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"
)

func decode(t *testing.T, s string) map[string]any {
	t.Helper()
	var raw map[string]any
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		t.Fatalf("bad test json %s: %v", s, err)
	}
	return raw
}

func at(x int) ZoneRef { return ZoneRef{Space: Coordinate{X: x}} }

func testView() (*fakeView, *Internals) {
	villager := UnitView{InternalID: 1, UnitID: 1, Type: UnitTypeEconomic, Kind: UnitEconomic, Location: at(0)}
	gatherer := UnitView{InternalID: 2, UnitID: 1, Type: UnitTypeEconomic, Kind: UnitEconomic, Location: at(0),
		CurrentOrder: &CurrentOrder{Kind: OrderKindGather},
		ActiveOrders: []ActiveUnitOrder{{ID: 50, Kind: OrderKindGather}, {ID: 51, Kind: OrderKindMove}}}
	infantry := UnitView{InternalID: 3, UnitID: 11, Type: UnitTypeInfantry, Kind: UnitMilitary, Location: at(0)}
	view := &fakeView{
		units: []UnitView{villager, gatherer, infantry},
		buildings: []BuildingView{
			{InternalID: 10, BuildingID: 1, Location: at(0), Health: 350, MaxHealth: 350, CanAttack: true, AutoAttack: true,
				ActiveOrders: []ActiveBuildingOrder{{ID: 60, Kind: BuildingOrderCreate, ItemID: 1}, {ID: 61, Kind: BuildingOrderResearch, ItemID: 1}, {ID: 62, Kind: BuildingOrderCreate, ItemID: 11}},
				Producibles:  []Producible{{Kind: ProducibleUnit, ID: 1, Cost: Cost{Food: 50}}}},
			{InternalID: 11, BuildingID: 2, Location: at(1), Health: 50, MaxHealth: 120},
			{InternalID: 12, BuildingID: 3, Location: at(1), Health: 120, MaxHealth: 120, Gatherable: true},
			{InternalID: 13, BuildingID: 22, Location: at(2), UnderConstruction: true, Producibles: []Producible{{Kind: ProducibleUnit, ID: 1, Cost: Cost{Food: 50}}}},
		},
		enemy_units:     []UnitView{{InternalID: 20, Kind: UnitMilitary, Type: UnitTypeInfantry, Location: at(1)}, {InternalID: 21, Kind: UnitMilitary, Type: UnitTypeInfantry, Location: at(5)}},
		enemy_buildings: []BuildingView{{InternalID: 30, Location: at(1)}},
		foundations:     []FoundationView{{BuildingID: 22, Location: at(3), Planned: true}, {BuildingID: 2, Location: at(4), Planned: true, Builders: 1}},
		resources:       map[ResourceCategory]float64{ResourceFood: 100, ResourceWood: 50},
		population:      3, limit: 10, turn: 12, num_players: 3, enemies_found: 2, land: 4, score: 500, best_enemy: 300,
		techs:     map[uint32]bool{1: true},
		available: map[ResourceCategory]bool{ResourceFood: true},
		in_range:  map[ZoneRef]bool{at(1): true},
		costs:     map[uint32]Cost{22: {Wood: 150, Stone: 50}, 2: {Wood: 30}},
	}
	internals := &Internals{}
	internals.Refresh()
	internals.spend(Cost{Wood: 20})
	army := internals.bucket("army")
	army.Desired = 2
	army.Members[1], army.Members[3] = true, true
	return view, internals
}

func TestConditions(t *testing.T) {
	cases := []struct {
		json string
		want bool
	}{
		{`{"always": {}}`, true},
		{`{"turn_at_least": {"amount": 12}}`, true},
		{`{"turn_at_most": {"amount": 11}}`, false},
		{`{"turn_equals": {"amount": 12}}`, true},
		{`{"not": {"turn_at_least": {"amount": 13}}}`, true},
		{`{"all": [{"turn_equals": {"amount": 12}}, {"land_at_least": {"amount": 5}}]}`, false},
		{`{"any": [{"turn_equals": {"amount": 1}}, {"land_equals": {"amount": 4}}]}`, true},
		{`{"num_players_equals": {"amount": 3}}`, true},
		{`{"enemies_found_equals": {"amount": "num_players - 1"}}`, true},
		{`{"enemies_found_at_most": {"amount": "num_players - 2"}}`, false},
		{`{"score_lead_equals": {"amount": 200}}`, true},
		{`{"score_lead_at_least": {"amount": "best_enemy_score"}}`, false},
		{`{"resource_at_least": {"category": "wood", "amount": 30}}`, true},
		{`{"resource_at_least": {"category": "wood", "amount": 31}}`, false},
		{`{"resource_at_least": {"category": "food", "amount": "population * 30"}}`, true},
		{`{"population_equals": {"amount": 3}}`, true},
		{`{"population_equals": {"unit_ids": [1], "amount": 2}}`, true},
		{`{"population_equals": {"unit_types": ["infantry"], "amount": 1}}`, true},
		{`{"population_equals": {"unit_ids": [11], "unit_types": ["economic"], "amount": 3}}`, true},
		{`{"population_headroom_equals": {"amount": 7}}`, true},
		{`{"idle_units_equals": {"amount": 2}}`, true},
		{`{"idle_units_equals": {"unit_types": ["economic"], "amount": 1}}`, true},
		{`{"enemy_units_visible_equals": {"amount": 2}}`, true},
		{`{"enemy_units_visible_equals": {"within": 2, "amount": 1}}`, true},
		{`{"building_count_equals": {"amount": 4}}`, true},
		{`{"building_count_equals": {"building_ids": [2, 3], "amount": 2}}`, true},
		{`{"building_count_equals": {"state": "complete", "amount": 3}}`, true},
		{`{"building_count_equals": {"state": "under_construction", "amount": 1}}`, true},
		{`{"building_count_equals": {"state": "damaged", "amount": 1}}`, true},
		{`{"building_count_equals": {"state": "depleted", "amount": 1}}`, true},
		{`{"foundation_count_equals": {"amount": 2}}`, true},
		{`{"foundation_count_equals": {"without_builders": true, "amount": 1}}`, true},
		{`{"foundation_count_equals": {"building_ids": [2], "without_builders": true, "amount": 0}}`, true},
		{`{"bucket_size_equals": {"bucket": "army", "amount": 2}}`, true},
		{`{"bucket_full": {"bucket": "army"}}`, true},
		{`{"bucket_full": {"bucket": "missing"}}`, false},
		{`{"tech_researched": {"tech_id": 1}}`, true},
		{`{"tech_researched": {"tech_id": 2}}`, false},
		{`{"resource_available": {"category": "food"}}`, true},
		{`{"resource_available": {"category": "stone"}}`, false},
	}
	for _, c := range cases {
		condition, err := parseCondition(decode(t, c.json))
		if err != nil {
			t.Errorf("%s: parse error %v", c.json, err)
			continue
		}
		view, internals := testView()
		if got := condition.Evaluate(view, internals); got != c.want {
			t.Errorf("%s: got %v want %v", c.json, got, c.want)
		}
	}
}

func TestConditionParseErrors(t *testing.T) {
	for _, s := range []string{
		`{"turn_at_least": {"amount": "bogus + 1"}}`,
		`{"turn_at_least": {}}`,
		`{"foo_at_least": {"amount": 1}}`,
		`{"turn_greater": {"amount": 1}}`,
		`{"building_count_equals": {"state": "weird", "amount": 1}}`,
		`{"building_count_equals": {"building_ids": 3, "amount": 1}}`,
		`{"population_equals": {"unit_types": ["wizard"], "amount": 1}}`,
		`{"resource_at_least": {"category": "gems", "amount": 1}}`,
		`{"tech_researched": {}}`,
		`{"bucket_full": {}}`,
	} {
		if _, err := parseCondition(decode(t, s)); err == nil {
			t.Errorf("%s: expected parse error", s)
		}
	}
}

func TestActionParsing(t *testing.T) {
	valid := []string{
		`{"action": "gather", "unit_types": ["economic"], "move_penalty": 2}`,
		`{"action": "gather", "category": "wood", "unit_ids": [1], "max": 2}`,
		`{"action": "create", "unit_id": 1}`,
		`{"action": "research", "tech_id": 1}`,
		`{"action": "build", "building_id": 2, "max": 2}`,
		`{"action": "buildNextInQ", "eligible": ["gather"], "weight": 7}`,
		`{"action": "createNextInQ"}`,
		`{"action": "researchNextInQ"}`,
		`{"action": "produce", "unit_types": ["economic"]}`,
		`{"action": "explore", "anchor": "home", "unit_types": ["infantry", "cavalry"]}`,
		`{"action": "attack", "target": "economic", "unit_ids": [11]}`,
		`{"action": "attack_space"}`,
		`{"action": "attack_zone"}`,
		`{"action": "garrison", "building_ids": [1], "unit_types": ["economic"]}`,
		`{"action": "ungarrison"}`,
		`{"action": "renew", "building_ids": [3]}`,
		`{"action": "repair", "building_ids": [1, 2], "max": 1}`,
		`{"action": "delete_unit", "unit_ids": [1]}`,
		`{"action": "delete_unit"}`,
		`{"action": "delete_building", "building_ids": [3]}`,
		`{"action": "delete_building"}`,
		`{"action": "building_attack", "target": "any", "building_ids": [1]}`,
		`{"action": "set_unit_behavior", "stance": "stand_ground", "attack_back": false, "target_priority": ["building", "military"]}`,
		`{"action": "set_building_behavior", "auto_attack": false, "interrupt_current": true, "target_priority": ["economic"]}`,
		`{"action": "unit_stop", "orders": ["gather", "move"], "unit_types": ["economic"]}`,
		`{"action": "building_stop", "orders": ["create"], "item_ids": [1], "building_ids": [1]}`,
		`{"action": "build_foundations", "building_ids": [22], "max": 2}`,
		`{"action": "cancel_foundations", "building_ids": [22]}`,
		`{"action": "set_bucket", "bucket": "a", "size": 3, "task": {"action": "attack"}}`,
		`{"action": "fill_bucket", "bucket": "a", "unit_types": ["infantry"]}`,
		`{"action": "run_bucket", "bucket": "a", "when_full": true}`,
		`{"action": "drain_bucket", "to": "a", "from": "any"}`,
		`{"action": "empty_bucket", "bucket": "a"}`,
		`{"action": "add_q", "type": "building", "id": 2, "weight": 3}`,
		`{"action": "add_q", "type": "resource", "cost": {"food": 50}}`,
	}
	for _, s := range valid {
		if _, err := parseAction(decode(t, s)); err != nil {
			t.Errorf("%s: %v", s, err)
		}
	}
	invalid := []string{
		`{"action": "empty_bucket", "bucket": "a", "unit_ids": [1]}`,
		`{"action": "delete_building", "unit_types": ["infantry"]}`,
		`{"action": "create", "unit_id": 1, "building_ids": [1]}`,
		`{"action": "attack", "building_ids": [1]}`,
		`{"action": "unit_stop", "orders": ["dance"]}`,
		`{"action": "building_stop", "orders": ["dance"]}`,
		`{"action": "set_unit_behavior", "stance": "sleepy"}`,
		`{"action": "set_building_behavior", "target_priority": ["trees"]}`,
		`{"action": "building_attack", "target": "trees"}`,
		`{"action": "gather", "unit_types": ["wizard"]}`,
	}
	for _, s := range invalid {
		if _, err := parseAction(decode(t, s)); err == nil {
			t.Errorf("%s: expected parse error", s)
		}
	}
}

func runAction(t *testing.T, view View, internals *Internals, s string) []string {
	t.Helper()
	action, err := parseAction(decode(t, s))
	if err != nil {
		t.Fatalf("%s: %v", s, err)
	}
	summary := make([]string, 0)
	for _, o := range action.ToOrders(view, internals) {
		summary = append(summary, fmt.Sprintf("%d:%v:%d", o.OrderType, o.Subjects, o.TargetID))
	}
	return summary
}

func expectOrders(t *testing.T, s string, got []string, want ...string) {
	t.Helper()
	if !slices.Equal(got, want) && !(len(got) == 0 && len(want) == 0) {
		t.Errorf("%s: got %v want %v", s, got, want)
	}
}

func TestActionOrders(t *testing.T) {
	cases := []struct {
		json  string
		setup func(v *fakeView)
		want  []string
	}{
		{`{"action": "repair"}`, nil, []string{"100:[1]:11"}},
		{`{"action": "repair"}`, func(v *fakeView) {
			v.units[0].CurrentOrder = &CurrentOrder{Kind: OrderKindRepair, TargetBuilding: &v.buildings[1]}
		}, nil},
		{`{"action": "delete_unit", "unit_types": ["infantry"]}`, nil, []string{"101:[3]:0"}},
		{`{"action": "delete_unit"}`, nil, []string{"101:[1]:0", "101:[3]:0"}},
		{`{"action": "delete_unit", "max": 1}`, nil, []string{"101:[1]:0"}},
		{`{"action": "delete_building", "building_ids": [2]}`, nil, []string{"102:[11]:0"}},
		{`{"action": "building_attack"}`, nil, nil},
		{`{"action": "building_attack"}`, func(v *fakeView) { v.buildings[0].ActiveOrders = nil }, []string{"103:[10]:20"}},
		{`{"action": "building_attack"}`, func(v *fakeView) { v.buildings[0].ActiveOrders = nil; v.enemy_units = v.enemy_units[1:] }, []string{"104:[10]:30"}},
		{`{"action": "unit_stop", "orders": ["gather"]}`, nil, []string{"105:[]:50"}},
		{`{"action": "unit_stop"}`, nil, []string{"105:[]:50", "105:[]:51"}},
		{`{"action": "unit_stop", "unit_types": ["infantry"]}`, nil, nil},
		{`{"action": "building_stop", "orders": ["create"], "item_ids": [1]}`, nil, []string{"105:[]:60"}},
		{`{"action": "building_stop", "orders": ["research"]}`, nil, []string{"105:[]:61"}},
		{`{"action": "building_stop", "building_ids": [2]}`, nil, nil},
		{`{"action": "building_stop"}`, nil, []string{"105:[]:60", "105:[]:61", "105:[]:62"}},
		{`{"action": "cancel_foundations"}`, nil, []string{"106:[]:22"}},
		{`{"action": "cancel_foundations", "building_ids": [2]}`, nil, nil},
		{`{"action": "build_foundations"}`, nil, []string{"107:[1]:22003"}},
		{`{"action": "build", "building_id": 22}`, nil, []string{"107:[1]:22003"}},
		{`{"action": "build", "building_id": 2}`, func(v *fakeView) { v.build_site = at(7) }, []string{"107:[1]:2007"}},
		{`{"action": "build", "building_id": 2}`, func(v *fakeView) { v.resources[ResourceWood] = 49 }, nil},
		{`{"action": "renew"}`, func(v *fakeView) { v.buildings[2].RenewCost = Cost{Wood: 30} }, []string{"108:[1]:12"}},
		{`{"action": "renew"}`, func(v *fakeView) { v.buildings[2].RenewCost = Cost{Wood: 31} }, nil},
		{`{"action": "renew"}`, func(v *fakeView) { v.buildings[2].RenewCost = Cost{Wood: 31}; v.buildings[2].Renewing = true }, []string{"108:[1]:12"}},
		{`{"action": "create", "unit_id": 1}`, nil, []string{"110:[10]:1", "110:[13]:1"}},
		{`{"action": "create", "unit_id": 1}`, func(v *fakeView) { v.limit = 4 }, []string{"110:[10]:1"}},
		{`{"action": "create", "unit_id": 1}`, func(v *fakeView) { v.resources[ResourceFood] = 99 }, []string{"110:[10]:1"}},
	}
	for _, c := range cases {
		view, internals := testView()
		if c.setup != nil {
			c.setup(view)
		}
		expectOrders(t, c.json, runAction(t, view, internals, c.json), c.want...)
	}
}

func TestInternalsEffects(t *testing.T) {
	view, internals := testView()
	runAction(t, view, internals, `{"action": "set_building_behavior", "auto_attack": true}`)
	runAction(t, view, internals, `{"action": "set_building_behavior", "auto_attack": false}`)
	if len(internals.building_behaviors) != 1 || !slices.Equal(internals.building_behaviors[0].Subjects, []uint64{10}) {
		t.Errorf("building behaviors: %+v", internals.building_behaviors)
	}
	runAction(t, view, internals, `{"action": "set_unit_behavior", "stance": "aggressive"}`)
	view.units[2].Stance = UnitStanceAggressive
	runAction(t, view, internals, `{"action": "set_unit_behavior", "stance": "aggressive"}`)
	if len(internals.behaviors) != 1 || !slices.Equal(internals.behaviors[0].Subjects, []uint64{3}) {
		t.Errorf("unit behaviors: %+v", internals.behaviors)
	}
	runAction(t, view, internals, `{"action": "empty_bucket", "bucket": "army"}`)
	if b := internals.Buckets["army"]; len(b.Members) != 0 || b.Desired != 2 {
		t.Errorf("empty_bucket: %+v", b)
	}
	runAction(t, view, internals, `{"action": "cancel_foundations"}`)
	if got := internals.available(view, ResourceWood); got != 180 {
		t.Errorf("cancel_foundations refund: wood available %v want 180", got)
	}
	view.build_site = at(7)
	runAction(t, view, internals, `{"action": "build", "building_id": 2}`)
	if got := internals.available(view, ResourceWood); got != 150 {
		t.Errorf("build spend: wood available %v want 150", got)
	}
}
