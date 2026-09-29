package ai

import "testing"

// Minimal View for expression tests: only the calls expressionVars and the population counter make
type fakeView struct {
	View
	turn            int
	food            float64
	units           []UnitView
	enemies         []UnitView
	enemy_buildings []BuildingView
	buildings       []BuildingView
}

func at(x, y int) ZoneRef { return ZoneRef{Space: Coordinate{X: x, Y: y}} }

func (f fakeView) NumPlayers() int        { return 2 }
func (f fakeView) EnemiesFound() int      { return 1 }
func (f fakeView) TurnNumber() int        { return f.turn }
func (f fakeView) OwnedSpaces() int       { return 5 }
func (f fakeView) Score() int             { return 10 }
func (f fakeView) BestEnemyScore() int    { return 8 }
func (f fakeView) Population() (int, int) { return len(f.units), 20 }
func (f fakeView) Resource(c ResourceCategory) float64 {
	return map[ResourceCategory]float64{ResourceFood: f.food}[c]
}
func (f fakeView) Units() []UnitView { return f.units }

func TestSetVarAndVarLookup(t *testing.T) {
	view := fakeView{turn: 7, food: 300, units: []UnitView{{UnitID: 1}, {UnitID: 1}, {UnitID: 13}}}
	internals := &Internals{}
	run := func(raw map[string]any) {
		t.Helper()
		action, err := parseAction(raw)
		if err != nil {
			t.Fatal(err)
		}
		action.ToOrders(view, internals)
	}
	run(map[string]any{"action": "set_var", "name": "goal", "value": "max(24 - var(population_1), 0)"})
	run(map[string]any{"action": "set_var", "name": "seen", "value": "max(var(seen), var(population_13) * 3)", "persist": true})
	run(map[string]any{"action": "set_var", "name": "sum", "value": "var(goal) + var(seen) + min(food, 100) + var(turn)"})
	check := func(name string, want float64) {
		t.Helper()
		if got := internals.lookupVar(view, name); got != want {
			t.Errorf("var(%s) = %v, want %v", name, got, want)
		}
	}
	check("goal", 22)
	check("seen", 3)
	check("sum", 22+3+100+7)
	check("population_1", 2)
	check("never_set", 0)
	internals.Refresh()
	check("goal", 0) // per-turn vars are cleared
	check("seen", 3) // globals persist
}

func TestBuiltinVarNames(t *testing.T) {
	good := []string{"population_13", "population_infantry", "population_11_12", "idle_units_1", "enemy_units_visible_13_within_2",
		"building_count_23_complete", "building_count_2_under_construction", "foundation_count_2_without_builders",
		"resource_remaining_wood_within_1", "bucket_size_vil_production", "tech_researched_5", "resource_available_gold",
		"land", "turn", "population_limit", "enemy_units_visible_1_within_1_of_raid", "population_infantry_within_2_of_vil_production",
		"enemy_buildings_visible_1_23_within_1_of_target", "enemy_buildings_visible", "resource_remaining_food_within_0_of_target",
		"target_distance", "target_distance_home"}
	for _, name := range good {
		if c, err := builtinCounter(name); err != nil || c == nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	for _, name := range []string{"populaton_13", "population_bogus", "enemy_units_visible_within", "tech_researched",
		"population_headroom", "score_lead", "resource_food", "enemy_units_visible_1_within_of_raid"} {
		if _, err := builtinCounter(name); err == nil {
			t.Errorf("%s should not resolve", name)
		}
	}
}

func TestExpressionInputs(t *testing.T) {
	for _, raw := range []map[string]any{
		{"action": "set_bucket", "bucket": "food", "size": "min(var(x), 6)", "task": map[string]any{"action": "gather", "category": "food"}},
		{"action": "add_q", "type": "unit", "id": 1.0, "weight": "max(0, 24 - var(population_1))"},
		{"action": "army", "launch": "var(enemy_seen) * 1.3", "strike": 12.0},
		{"action": "build", "building_id": 2.0, "max": "1 + var(x)"},
		{"action": "create", "unit_id": 1.0, "queue": "2"},
	} {
		if _, err := parseAction(raw); err != nil {
			t.Errorf("%v: %v", raw["action"], err)
		}
	}
	for _, raw := range []map[string]any{
		{"action": "build", "building_id": 2.0, "max": "1 +"},
		{"action": "set_var", "name": "9bad", "value": 1.0},
		{"action": "set_var", "name": "x", "value": "min(1)"},
	} {
		if _, err := parseAction(raw); err == nil {
			t.Errorf("%v should fail to parse", raw)
		}
	}
	cond, err := parseCondition(map[string]any{"value_at_least": map[string]any{"value": "var(x) * 2", "amount": 4.0}})
	if err != nil {
		t.Fatal(err)
	}
	internals := &Internals{turn_vars: map[string]float64{"x": 2}}
	if !cond.Evaluate(fakeView{}, internals) {
		t.Errorf("value_at_least should pass with x=2")
	}
}

func TestAnchorsAndTargetPicker(t *testing.T) {
	raiders := []UnitView{{InternalID: 1, UnitID: 11, Kind: UnitMilitary, Location: at(2, 0)}, {InternalID: 2, UnitID: 11, Kind: UnitMilitary, Location: at(2, 0)}}
	view := fakeView{
		units:     raiders,
		buildings: []BuildingView{{BuildingID: 1, Location: at(-3, 0)}},
		enemies: []UnitView{
			// a lone villager, a defended one next to a Village Center, and five together
			{InternalID: 10, UnitID: 1, Kind: UnitEconomic, Location: at(3, 0)},
			{InternalID: 11, UnitID: 1, Kind: UnitEconomic, Location: at(5, 0)},
			{InternalID: 12, UnitID: 1, Kind: UnitEconomic, Location: at(2, 2)}, {InternalID: 13, UnitID: 1, Kind: UnitEconomic, Location: at(2, 2)},
			{InternalID: 14, UnitID: 1, Kind: UnitEconomic, Location: at(2, 2)}, {InternalID: 15, UnitID: 1, Kind: UnitEconomic, Location: at(2, 2)},
			{InternalID: 16, UnitID: 1, Kind: UnitEconomic, Location: at(2, 2)},
		},
		enemy_buildings: []BuildingView{{BuildingID: 1, Location: at(5, 1)}},
	}
	internals := &Internals{Buckets: map[string]*Bucket{"raid": {Members: map[uint64]bool{1: true, 2: true}}}}
	if got := internals.lookupVar(view, "enemy_units_visible_1_within_2_of_raid"); got != 6 {
		t.Errorf("villagers within 2 of the raid = %v, want 6", got)
	}
	if got := internals.lookupVar(view, "population_11_within_0_of_raid"); got != 2 {
		t.Errorf("raiders at the raid center = %v, want 2", got)
	}
	attack, err := parseAction(map[string]any{"action": "attack", "in_bucket": "raid", "targets": "enemy_units", "target_unit_ids": []any{1.0},
		"score":     "10 - var(target_distance) - 20 * var(enemy_buildings_visible_1_within_1_of_target) - 3 * var(enemy_units_visible_1_within_0_of_target)",
		"min_score": 0.0, "together": true})
	if err != nil {
		t.Fatal(err)
	}
	orders := attack.ToOrders(view, internals)
	if len(orders) != 2 || orders[0].TargetID != 10 || orders[1].TargetID != 10 {
		t.Fatalf("raid should pick the lone villager 10 for both units, got %+v", orders)
	}
	space_attack, err := parseAction(map[string]any{"action": "attack", "in_bucket": "raid", "targets": "enemy_units", "target_unit_ids": []any{1.0},
		"score": "10 - var(target_distance)", "together": true, "order": "space"})
	if err != nil {
		t.Fatal(err)
	}
	if orders := space_attack.ToOrders(view, internals); len(orders) != 2 || orders[0].OrderType != 3 || orders[0].TargetID != 300 {
		t.Errorf("order space should attack the target's space (3,0), got %+v", orders)
	}
	if _, err := parseAction(map[string]any{"action": "attack", "targets": "enemy_units", "order": "sideways"}); err == nil {
		t.Errorf("unknown order should fail")
	}
	// nothing clears min_score: no orders
	strict, _ := parseAction(map[string]any{"action": "attack", "targets": "enemy_units", "score": "0 - var(target_distance)", "min_score": 0.0})
	if orders := strict.ToOrders(view, internals); len(orders) != 0 {
		t.Errorf("no candidate reaches min_score, got %+v", orders)
	}
	move, err := parseAction(map[string]any{"action": "move", "in_bucket": "raid", "targets": "home", "together": true})
	if err != nil {
		t.Fatal(err)
	}
	if orders := move.ToOrders(view, internals); len(orders) != 2 || orders[0].TargetID != -300 {
		t.Errorf("retreat should move both raiders home, got %+v", orders)
	}
	if internals.target != nil {
		t.Errorf("picker must restore the target context")
	}
	if _, err := parseAction(map[string]any{"action": "move"}); err == nil {
		t.Errorf("move without targets should fail")
	}
}

func (f fakeView) VisibleEnemyUnits() []UnitView         { return f.enemies }
func (f fakeView) VisibleEnemyBuildings() []BuildingView { return f.enemy_buildings }
func (f fakeView) Buildings() []BuildingView             { return f.buildings }
func (f fakeView) IdleUnits() []UnitView                 { return f.units }
func (f fakeView) EligibleUnits(...OrderKind) []UnitView { return f.units }
func (f fakeView) AttackUnitOrder(u UnitView, t UnitView, _ bool) Order {
	return Order{Subjects: []uint64{u.InternalID}, TargetID: int64(t.InternalID), OrderType: 1}
}
func (f fakeView) AttackSpaceOrder(u UnitView, c Coordinate, _ bool) Order {
	return Order{Subjects: []uint64{u.InternalID}, TargetID: int64(c.X*100 + c.Y), OrderType: 3}
}
func (f fakeView) MoveOrder(u UnitView, z ZoneRef, _ bool) Order {
	return Order{Subjects: []uint64{u.InternalID}, TargetID: int64(z.Space.X*100 + z.Space.Y), OrderType: 2}
}
