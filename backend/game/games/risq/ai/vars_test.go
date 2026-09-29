package ai

import "testing"

// Minimal View for expression tests: only the calls expressionVars and the population counter make
type fakeView struct {
	View
	turn  int
	food  float64
	units []UnitView
}

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
		"building_count_23_complete", "building_count_2_under_construction", "foundation_count_2_without_builders", "resource_food",
		"resource_remaining_wood_within_1", "bucket_size_vil_production", "tech_researched_5", "resource_available_gold",
		"population_headroom", "land", "turn", "population_limit"}
	for _, name := range good {
		if c, err := builtinCounter(name); err != nil || c == nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	for _, name := range []string{"populaton_13", "population_bogus", "enemy_units_visible_within", "tech_researched"} {
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
