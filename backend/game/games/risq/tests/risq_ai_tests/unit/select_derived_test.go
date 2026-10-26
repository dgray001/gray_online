package unit

import (
	"encoding/json"
	"io"
	"math/rand"
	"testing"

	. "github.com/dgray001/gray_online/game/games/risq/ai"
)

func lineOfSpaces(gold ...float64) []SpaceInfo {
	spaces := make([]SpaceInfo, len(gold))
	for i, g := range gold {
		spaces[i] = SpaceInfo{Space: Coordinate{X: i + 1}, GoldIncome: g, Vision: 3, Owner: -1}
	}
	return spaces
}

func TestSetLookupReplacementAndDecisionLifetime(t *testing.T) {
	view := selectTestView{loopTestView: loopTestView{units: []UnitView{{InternalID: 1}, {InternalID: 2}}}, turn: 1}
	config := `{"rules":[
		{"when":{"value_equals":{"value":"gameVar(turn)","amount":1}},"then":[{"action":"select","pick":{"kind":"units"},"into":{"set":"kept","persist":true}}]},
		{"when":{"always":{}},"then":[{"action":"set_var","name":"before","value":{"count":{"kind":"units","where":{"in_set":"kept"}}}}]},
		{"when":{"value_equals":{"value":"gameVar(turn)","amount":1}},"then":[{"action":"select","pick":{"kind":"units","where":{"in":{"uid":[2]}}},"into":{"set":"kept"}}]},
		{"when":{"always":{}},"then":[{"action":"set_var","name":"after","value":{"sum":{"kind":"units","where":{"in_set":"kept"},"of":"var(item.uid)"}}},
		{"action":"set_var","name":"excluded","value":{"sum":{"kind":"units","where":{"not_in_set":"kept"},"of":"var(item.uid)"}}}]}]}`
	model := parseSelectModel(t, config)
	model.DecideOrders(view)
	assertLookupValues(t, model, 2, 2, 1)
	view.turn = 2
	model.DecideOrders(view)
	assertLookupValues(t, model, 0, 0, 3)
}

func assertLookupValues(t *testing.T, model *RulesModel, before float64, after float64, excluded float64) {
	t.Helper()
	for name, want := range map[string]float64{"before": before, "after": after, "excluded": excluded} {
		if got, _ := model.Var(name); got != want {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
}

func TestSelectRejectsLegacySpatialFiltersAndMalformedCoordinates(t *testing.T) {
	for _, where := range []string{`{"within":3}`, `{"from":"home"}`, `{"within":3,"from":"army"}`, `{"near":{"of":{"kind":"units","where":{"within":1}}}}`} {
		if err := loadError(rule(`{"action":"select","pick":{"kind":"spaces","where":` + where + `},"into":{"set":"x"}}`)); err == nil {
			t.Errorf("legacy spatial filter accepted: %s", where)
		}
	}
	for _, origin := range []string{`{}`, `{"x":1}`, `{"x":1,"y":null}`, `{"x":true,"y":0}`, `{"x":1,"y":0,"zone_x":1}`, `{"x":1,"y":0,"extra":1}`, `{"kind":"spaces","x":1,"y":0}`} {
		if err := loadError(rule(`{"action":"select","pick":{"kind":"spaces","where":{"near":{"of":` + origin + `}}},"into":{"set":"x"}}`)); err == nil {
			t.Errorf("malformed coordinate accepted: %s", origin)
		}
	}
}

func TestSelectEmptyBucketModesPreserveOtherMemberships(t *testing.T) {
	view := selectTestView{loopTestView: loopTestView{units: []UnitView{{InternalID: 1}, {InternalID: 2}}}}
	for mode, want := range map[string]float64{"": 0, `,"mode":"replace"`: 0, `,"mode":"add"`: 2, `,"mode":"remove"`: 2} {
		config := rule(`{"action":"select","pick":{"kind":"units"},"into":{"bucket":"other"}},
			{"action":"select","pick":{"kind":"units"},"into":{"bucket":"group"}},
			{"action":"select","pick":{"kind":"units","where":{"in":{"uid":[99]}}},"into":{"bucket":"group"` + mode + `}},
			{"action":"set_var","name":"group_n","value":{"count":{"kind":"units","where":{"bucket":"group"}}}},
			{"action":"set_var","name":"other_n","value":{"count":{"kind":"units","where":{"bucket":"other"}}}}`)
		vars := decide(t, config, view, "group_n", "other_n")
		if vars["group_n"] != want || vars["other_n"] != 2 {
			t.Errorf("mode %q: counts %v, want group %v and other 2", mode, vars, want)
		}
	}
}

func TestSelectSetKindsPreserveMatchingAndEmptySets(t *testing.T) {
	worker := villager(1, Coordinate{})
	view := selectTestView{loopTestView: loopTestView{units: []UnitView{worker}}, idle: []UnitView{worker}}
	for _, limit := range []string{"0", "1"} {
		config := rule(`{"action":"select","pick":{"kind":"units","limit":` + limit + `},"into":{"set":"chosen","persist":true,"var":"size"}},
			{"action":"set_var","name":"matching","value":{"count":{"kind":"units","where":{"in_set":"chosen"}}}},
			{"action":"set_var","name":"excluded","value":{"count":{"kind":"units","where":{"not_in_set":"chosen"}}}},
			{"action":"set_var","name":"mismatch","value":{"count":{"kind":"buildings","where":{"not_in_set":"chosen"}}}}`)
		vars := decide(t, config, view, "size", "matching", "excluded", "mismatch")
		if vars["matching"] != vars["size"] || vars["excluded"] != 1-vars["size"] || vars["mismatch"] != 0 {
			t.Errorf("limit %s: unexpected counts %v", limit, vars)
		}
	}
}

func moveTarget(t *testing.T, config string, view View) int64 {
	t.Helper()
	orders := selectOrders(t, config, view)
	if len(orders) != 1 || orders[0].OrderType != kindMoveSpace {
		t.Fatalf("orders = %+v, want one space move", orders)
	}
	return orders[0].TargetID
}

func TestAggregatesOverSelectors(t *testing.T) {
	view := selectTestView{loopTestView: loopTestView{spaces: lineOfSpaces(4, 9, 7, 1)}, turn: 1}
	config := rule(`{"action":"set_var","name":"total","value":{"sum":{"kind":"spaces","of":"var(item.gold_income)"}}},
		{"action":"set_var","name":"lowest","value":{"min":{"kind":"spaces","of":"var(item.gold_income)"}}},
		{"action":"set_var","name":"highest","value":{"max":{"kind":"spaces","of":"var(item.gold_income)"}}},
		{"action":"set_var","name":"rich","value":{"count":{"kind":"spaces","where":{"if":"var(item.gold_income) - 5"}}}},
		{"action":"set_var","name":"top_two","value":{"sum":{"kind":"spaces","rank":"var(item.gold_income)","limit":2,"of":"var(item.gold_income)"}}}`)
	vars := decide(t, config, view, "total", "lowest", "highest", "rich", "top_two")
	want := map[string]float64{"total": 21, "lowest": 1, "highest": 9, "rich": 2, "top_two": 16}
	for name, value := range want {
		if vars[name] != value {
			t.Errorf("%s = %v, want %v", name, vars[name], value)
		}
	}
}

func TestSelectorFindsNothingWhenItsStepBudgetRunsOut(t *testing.T) {
	config := rule(`{"action":"select","into":{"set":"x","var":"picked"},
		"pick":{"kind":"spaces","limit":1,"rank":"var(target.all)","let":{"all":{"count":{"kind":"spaces"}}}}}`)
	for spaces, want := range map[int]float64{10: 1, 800: 0} {
		view := selectTestView{loopTestView: loopTestView{spaces: lineOfSpaces(make([]float64, spaces)...)}, turn: 1}
		if got := decide(t, config, view, "picked")["picked"]; got != want {
			t.Errorf("%d spaces ranked by a count over all of them picked %v, want %v", spaces, got, want)
		}
	}
}

func TestInSetSelectorReturnsExactlyTheMembers(t *testing.T) {
	units := make([]UnitView, 20)
	for n := range units {
		units[n] = villager(uint64(n+1), Coordinate{})
	}
	view := selectTestView{loopTestView: loopTestView{units: units, spaces: lineOfSpaces(make([]float64, 9)...)}, turn: 1}
	sums := func(kind string, set string, field string) string {
		return `{"sum":{"kind":"` + kind + `","where":{"in_set":"` + set + `"},"of":"var(item.` + field + `)"}}`
	}
	config := rule(`{"action":"select","pick":{"kind":"units","limit":3,"rank":"0 - var(target.uid)"},"into":{"set":"units_kept"}},
		{"action":"select","pick":{"kind":"spaces","where":{"if":"var(target.x) <= 2"}},"into":{"set":"spaces_kept"}},
		{"action":"set_var","name":"unit_sum","value":` + sums("units", "units_kept", "uid") + `},
		{"action":"set_var","name":"space_sum","value":` + sums("spaces", "spaces_kept", "x") + `},
		{"action":"set_var","name":"unknown_set","value":{"count":{"kind":"units","where":{"in_set":"never_stored"}}}}`)
	got := decide(t, config, view, "unit_sum", "space_sum", "unknown_set")
	if got["unit_sum"] != 6 || got["space_sum"] != 3 || got["unknown_set"] != 0 {
		t.Errorf("sums over stored sets = %v, want units 6, spaces 3 and nothing from a set never stored", got)
	}
}

func TestMinAndMaxDefaultOnlyAppliesToAnEmptySet(t *testing.T) {
	view := selectTestView{loopTestView: loopTestView{spaces: lineOfSpaces(4, 9, 7, 1)}, turn: 1}
	none := `"where":{"if":"var(item.gold_income) - 100"}`
	config := rule(`{"action":"set_var","name":"bare","value":{"min":{"kind":"spaces",` + none + `,"of":"var(item.gold_income)"}}},
		{"action":"set_var","name":"empty_min","value":{"min":{"kind":"spaces",` + none + `,"of":"var(item.gold_income)","default":99}}},
		{"action":"set_var","name":"empty_max","value":{"max":{"kind":"spaces",` + none + `,"of":"var(item.gold_income)","default":"10 * 2"}}},
		{"action":"set_var","name":"filled","value":{"min":{"kind":"spaces","of":"var(item.gold_income)","default":99}}}`)
	got := decide(t, config, view, "bare", "empty_min", "empty_max", "filled")
	want := map[string]float64{"bare": 0, "empty_min": 99, "empty_max": 20, "filled": 1}
	for name, value := range want {
		if got[name] != value {
			t.Errorf("%s = %v, want %v", name, got[name], value)
		}
	}
	for _, name := range []string{"count", "sum"} {
		action := `{"action":"set_var","name":"x","value":{"` + name + `":{"kind":"spaces","of":"var(item.gold_income)","default":1}}}`
		if name == "count" {
			action = `{"action":"set_var","name":"x","value":{"count":{"kind":"spaces","default":1}}}`
		}
		if loadError(rule(action)) == nil {
			t.Errorf("%s accepted a default", name)
		}
	}
}

func TestAggregateOfCanReadItsOwnLets(t *testing.T) {
	view := selectTestView{loopTestView: loopTestView{spaces: lineOfSpaces(4, 9, 7, 1)}, turn: 1}
	config := rule(`{"action":"set_var","name":"doubled","value":{"sum":{"kind":"spaces","let":{"double":"2 * var(item.gold_income)"},"of":"var(item.double)"}}}`)
	if got := decide(t, config, view, "doubled")["doubled"]; got != 42 {
		t.Errorf("sum of doubled incomes = %v, want 42", got)
	}
}

func TestEnemyDistanceDerivedFromABuildingAggregate(t *testing.T) {
	worker := villager(1, Coordinate{})
	view := selectTestView{
		loopTestView: loopTestView{
			units:           []UnitView{worker},
			spaces:          lineOfSpaces(0, 0, 0, 0),
			enemy_buildings: []BuildingView{{InternalID: 31, PlayerID: 1, Location: ZoneRef{Space: Coordinate{X: 5}}}},
		},
		idle: []UnitView{worker}, turn: 1,
	}
	pick := func(rank string) string {
		return rule(`{"action":"select","pick":{"kind":"spaces","limit":1,"rank":"` + rank + `",
			"let":{"enemy_dist":{"min":{"kind":"buildings","where":{"owner":"enemy"},"of":"var(item.distance_to_target)"}}}},
			"issue":[{"move":{}}]}`)
	}
	if got := moveTarget(t, pick("var(target.enemy_dist)"), view); got != 100 {
		t.Errorf("farthest from the enemy = %d, want the space at x=1", got)
	}
	if got := moveTarget(t, pick("0 - var(target.enemy_dist)"), view); got != 400 {
		t.Errorf("closest to the enemy = %d, want the space at x=4", got)
	}
}

func regionView() selectTestView {
	spaces := []SpaceInfo{
		{Space: Coordinate{X: 1}, Owner: 0, Vision: 3}, {Space: Coordinate{X: 2}, Owner: -1, Vision: 3},
		{Space: Coordinate{X: 3}, Owner: 1, Vision: 3}, {Space: Coordinate{X: 4}, Owner: -1, Vision: 1},
	}
	regions := []RegionInfo{
		{Name: "west", GoldBonus: 10, Owner: -1, Spaces: []Coordinate{{X: 1}, {X: 2}}},
		{Name: "east", GoldBonus: 3, Owner: 1, Spaces: []Coordinate{{X: 3}}, HasUnexplored: true},
	}
	return selectTestView{loopTestView: loopTestView{spaces: spaces, regions: regions}, turn: 1}
}

func TestPersistentRegionSetKeepsIdentityAfterDiscovery(t *testing.T) {
	worker := villager(1, Coordinate{})
	view := selectTestView{loopTestView: loopTestView{units: []UnitView{worker}, spaces: lineOfSpaces(0, 0), regions: []RegionInfo{{Name: "kept", Spaces: []Coordinate{{X: 2}}}}}, idle: []UnitView{worker}, turn: 1}
	config := `{"rules":[{"when":{"value_equals":{"value":"gameVar(turn)","amount":1}},"then":[{"action":"select","pick":{"kind":"regions"},"into":{"set":"kept","persist":true}}]},{"when":{"always":{}},"then":[{"action":"select","pick":{"kind":"spaces","where":{"region_in_set":"kept"}},"issue":[{"move":{}}]}]}]}`
	model := parseSelectModel(t, config)
	model.DecideOrders(view)
	view.turn = 2
	view.regions = append([]RegionInfo{{Name: "new", Spaces: []Coordinate{{X: 1}}}}, view.regions...)
	orders := model.DecideOrders(view).Orders
	if len(orders) != 1 || orders[0].TargetID != spaceCode(Coordinate{X: 2}) {
		t.Fatalf("turn 2 orders = %+v, want persisted region kept at x=2", orders)
	}
}

func TestRegionFieldsOnSpacesAndRegionProgress(t *testing.T) {
	worker := villager(1, Coordinate{})
	view := regionView()
	view.units, view.idle = []UnitView{worker}, []UnitView{worker}
	config := rule(`{"action":"select","pick":{"kind":"spaces","limit":1,"where":{"if":"max(var(target.owner), 0 - var(target.owner))"},
		"rank":"(var(target.region_owned) + 1) / var(target.region_known) + var(target.region_gold_bonus) / 100"},
		"issue":[{"move":{}}]}`)
	if got := moveTarget(t, config, view); got != 200 {
		t.Errorf("best space by region progress = %d, want the unowned space of the half-owned region (x=2)", got)
	}
}

func TestRegionsAreASelectableKindAndFilterOtherKinds(t *testing.T) {
	config := rule(`{"action":"select","pick":{"kind":"regions","where":{"if":"var(target.owned_spaces)"}},"into":{"set":"mine","var":"mine_n"}},
		{"action":"set_var","name":"in_mine","value":{"count":{"kind":"spaces","where":{"region_in_set":"mine"}}}},
		{"action":"set_var","name":"unexplored","value":{"count":{"kind":"regions","where":{"if":"var(item.has_unexplored)"}}}}`)
	vars := decide(t, config, regionView(), "mine_n", "in_mine", "unexplored")
	if vars["mine_n"] != 1 || vars["in_mine"] != 2 || vars["unexplored"] != 1 {
		t.Errorf("regions I own part of %v, spaces inside them %v, regions with unexplored spaces %v; want 1, 2, 1", vars["mine_n"], vars["in_mine"], vars["unexplored"])
	}
}

func TestRegionItemFieldsInForEach(t *testing.T) {
	config := rule(`{"action":"for_each","source":"regions","as":"r","do":[{"action":"set_var","name":"sum","value":"var(sum) + var(r.known_spaces) * 1000 + var(r.owned_spaces) * 100 + var(r.enemy_spaces) * 10 + var(r.unowned_spaces)","persist":true}]}`)
	if got := decide(t, config, regionView(), "sum")["sum"]; got != 3111 {
		t.Errorf("region field sum = %v, want 3111", got)
	}
}

func TestAssignedUnitsDerivedFromUnitOrderFields(t *testing.T) {
	attacking := func(id uint64, target uint64) UnitView {
		return UnitView{InternalID: id, CurrentOrder: &CurrentOrder{Kind: OrderKindAttackBuilding, TargetBuilding: &BuildingView{InternalID: target}}}
	}
	view := selectTestView{
		loopTestView: loopTestView{
			units: []UnitView{attacking(1, 21), attacking(2, 21), attacking(3, 22), {InternalID: 4, CanAttack: true}},
			enemy_buildings: []BuildingView{
				{InternalID: 21, PlayerID: 1, Location: ZoneRef{Space: Coordinate{X: 1}}},
				{InternalID: 22, PlayerID: 1, Location: ZoneRef{Space: Coordinate{X: 2}}},
			},
		},
		turn: 1,
	}
	config := rule(`{"action":"set_var","name":"on_21","value":{"count":{"kind":"units","where":{"if":"1 - max(var(item.order_target_uid) - 1000000021, 1000000021 - var(item.order_target_uid))"}}}},
		{"action":"set_var","name":"idle_orderless","value":{"count":{"kind":"units","where":{"if":"1 - var(item.has_order)"}}}}`)
	vars := decide(t, config, view, "on_21", "idle_orderless")
	if vars["on_21"] != 2 || vars["idle_orderless"] != 1 {
		t.Errorf("units on building 21 = %v, without orders = %v; want 2 and 1", vars["on_21"], vars["idle_orderless"])
	}
	spread := rule(`{"action":"select","pick":{"kind":"buildings","where":{"owner":"enemy"},"limit":1,
		"let":{"assigned":{"count":{"kind":"units","where":{"if":"1 - max(var(item.order_target_uid) - var(target.uid), var(target.uid) - var(item.order_target_uid))"}}}},
		"rank":"0 - var(target.assigned)"},"into":{"set":"least_attacked"}},
		{"action":"select","pick":{"kind":"buildings","where":{"in_set":"least_attacked"}},"actors":{"kind":"units","where":{"if":"1 - var(actor.has_order)"}},"issue":[{"attack":{}}]}`)
	orders := selectOrders(t, spread, view)
	if len(orders) != 1 || orders[0].OrderType != kindAttackBuilding || orders[0].TargetID != 22 {
		t.Fatalf("orders = %+v, want the idle unit sent against building 22 (one attacker instead of two)", orders)
	}
}

func TestSpaceVisionAvailableOnZones(t *testing.T) {
	worker := villager(1, Coordinate{})
	view := selectTestView{
		loopTestView: loopTestView{
			units:  []UnitView{worker},
			spaces: []SpaceInfo{{Space: Coordinate{X: 1}, Vision: 1}, {Space: Coordinate{X: 2}, Vision: 3}},
			zones: []ZoneInfo{
				{Location: ZoneRef{Space: Coordinate{X: 1}}, BuildingPlayer: -1}, {Location: ZoneRef{Space: Coordinate{X: 2}}, BuildingPlayer: -1},
			},
		},
		idle: []UnitView{worker}, turn: 1,
	}
	config := rule(`{"action":"select","pick":{"kind":"zones","limit":1,"rank":"var(target.space_vision)"},"issue":[{"move":{}}]}`)
	orders := selectOrders(t, config, view)
	if len(orders) != 1 || orders[0].OrderType != kindMoveZone || orders[0].TargetID != 200 {
		t.Fatalf("orders = %+v, want a zone move to the better seen space (x=2)", orders)
	}
}

func TestSelectorCountsAcceptVariablesAndAggregates(t *testing.T) {
	view := selectTestView{loopTestView: loopFixture()}
	config := rule(`{"action":"set_var","name":"offset","value":1},
		{"action":"select","pick":{"kind":"units","let":{"score":"var(target.health)"},"where":{"if":"var(target.score)"},"rank":"var(target.score)","skip":"var(offset)","limit":{"count":{"kind":"units","where":{"if":"var(item.health) - 5"}}}},"into":{"set":"chosen","var":"n"}},
		{"action":"set_var","name":"ids","value":{"sum":{"kind":"units","where":{"in_set":"chosen"},"of":"var(item.uid)"}}}`)
	got := decide(t, config, view, "n", "ids")
	if got["n"] != 2 || got["ids"] != 3 {
		t.Fatalf("selection = %v, want two units with ids summing to three", got)
	}
}

func TestSelectorCountsAndSourcesKeepEnclosingLoopBindings(t *testing.T) {
	for _, name := range []string{"u", "target"} {
		config := rule(`{"action":"set_var","name":"total","value":0},
			{"action":"for_each","source":"units","as":"` + name + `","do":[
			{"action":"select","pick":{"kind":"units","where":{"player_id":"var(` + name + `.player)"},"rank":"var(target.health)","skip":"var(` + name + `.uid) - 1","limit":1},"into":{"set":"chosen","var":"n"}},
			{"action":"set_var","name":"total","value":"var(total) + var(n)"}]}`)
		view := selectTestView{loopTestView: loopFixture()}
		if got := decide(t, config, view, "total")["total"]; got != 3 {
			t.Errorf("loop %s: total = %v, want three selections of one unit", name, got)
		}
	}
}

func TestNestedAggregateControlsReadTheOuterItem(t *testing.T) {
	config := rule(`{"action":"set_var","name":"total","value":{"sum":{"kind":"units","of":{"max":{"kind":"units","where":{"player_id":"var(item.player)"},"rank":"var(item.health)","skip":"var(item.uid) - 1","limit":"var(item.uid)","of":"var(item.health)"}}}}}`)
	view := selectTestView{loopTestView: loopFixture()}
	if got := decide(t, config, view, "total")["total"]; got != 19 {
		t.Errorf("nested total = %v, want 9 + 6 + 4", got)
	}
}

func TestLetAndAggregateConfigErrors(t *testing.T) {
	cases := map[string]string{
		"candidate limit":        `{"action":"select","pick":{"kind":"units","limit":"var(target.uid)"}}`,
		"candidate skip":         `{"action":"select","pick":{"kind":"units","skip":"var(target.uid)"}}`,
		"candidate let limit":    `{"action":"select","pick":{"kind":"units","let":{"n":2},"limit":"var(target.n)"}}`,
		"actor limit":            `{"action":"select","actors":{"kind":"units","limit":"var(actor.uid)"},"issue":[{"delete":{}}]}`,
		"actor source":           `{"action":"select","actors":{"kind":"units","where":{"player_id":"var(actor.player)"}},"issue":[{"delete":{}}]}`,
		"target source":          `{"action":"select","pick":{"kind":"units","where":{"player_id":"var(target.player)"}}}`,
		"space source":           `{"action":"select","pick":{"kind":"spaces","where":{"player_id":"var(target.owner)"}}}`,
		"space building source":  `{"action":"select","pick":{"kind":"spaces","where":{"building_player_id":"var(target.owner)"}}}`,
		"aggregate limit":        `{"action":"set_var","name":"x","value":{"sum":{"kind":"units","limit":"var(item.health)","of":"1"}}}`,
		"aggregate source":       `{"action":"set_var","name":"x","value":{"count":{"kind":"units","where":{"player_id":"var(item.player)"}}}}`,
		"let shadows a field":    `{"action":"select","pick":{"kind":"spaces","let":{"vision":1}},"into":{"set":"x"}}`,
		"let bad name":           `{"action":"select","pick":{"kind":"spaces","let":{"a b":1}},"into":{"set":"x"}}`,
		"let not an object":      `{"action":"select","pick":{"kind":"spaces","let":[1]},"into":{"set":"x"}}`,
		"unknown name in rank":   `{"action":"select","pick":{"kind":"spaces","rank":"var(target.nobody)"},"into":{"set":"x"}}`,
		"sum without of":         `{"action":"set_var","name":"x","value":{"sum":{"kind":"spaces"}}}`,
		"count with of":          `{"action":"set_var","name":"x","value":{"count":{"kind":"spaces","of":"1"}}}`,
		"aggregate unknown kind": `{"action":"set_var","name":"x","value":{"count":{"kind":"ships"}}}`,
		"aggregate bad field":    `{"action":"set_var","name":"x","value":{"max":{"kind":"spaces","of":"var(item.nope)"}}}`,
		"regions as verb target": `{"action":"select","pick":{"kind":"regions"},"issue":[{"move":{}}]}`,
	}
	for name, action := range cases {
		if loadError(rule(action)) == nil {
			t.Errorf("%s: malformed config accepted", name)
		}
	}
}

func TestSelfReferencingLetDoesNotHang(t *testing.T) {
	config := rule(`{"action":"select","pick":{"kind":"spaces","let":{"loop":"var(target.loop) + 1"},"rank":"var(target.loop)"},"into":{"set":"x","var":"n"}}`)
	view := selectTestView{loopTestView: loopTestView{spaces: lineOfSpaces(1, 2)}, turn: 1}
	if got := decide(t, config, view, "n")["n"]; got != 0 {
		t.Errorf("selected %v spaces, want none after the self-referencing let fails", got)
	}
}

func gatherNodeView(farm_left float64) selectTestView {
	worker := villager(1, Coordinate{})
	farm := BuildingView{InternalID: 30, BuildingID: 3, Location: ZoneRef{Space: Coordinate{X: 1}}, Gatherable: true, ResourceCategory: ResourceFood, ResourcesLeft: farm_left, GatherSpeed: 8, GatherCapacity: 2}
	unfinished := farm
	unfinished.InternalID, unfinished.UnderConstruction = 31, true
	bush := ResourceView{InternalID: 7, Category: ResourceFood, Location: ZoneRef{Space: Coordinate{X: 3}}, AmountLeft: 50, GatherSpeed: 12, GatherCapacity: 4}
	depleted := ResourceView{InternalID: 8, Category: ResourceFood, Location: ZoneRef{Space: Coordinate{X: 4}}, GatherSpeed: 12, GatherCapacity: 4}
	return selectTestView{
		loopTestView: loopTestView{units: []UnitView{worker}, buildings: []BuildingView{farm, unfinished}, resources: map[ResourceCategory][]ResourceView{ResourceFood: {bush, depleted}}},
		idle:         []UnitView{worker}, turn: 1,
	}
}

func TestGatherablesMergeResourcesAndOwnFinishedBuildings(t *testing.T) {
	config := rule(`{"action":"set_var","name":"nodes","value":{"count":{"kind":"gatherables"}}},
		{"action":"set_var","name":"capacity","value":{"sum":{"kind":"gatherables","of":"var(item.gather_capacity)"}}},
		{"action":"set_var","name":"buildings","value":{"sum":{"kind":"gatherables","of":"var(item.is_building)"}}}`)
	got := decide(t, config, gatherNodeView(100), "nodes", "capacity", "buildings")
	if got["nodes"] != 2 || got["capacity"] != 6 || got["buildings"] != 1 {
		t.Errorf("nodes, capacity, buildings = %v, want 2, 6, 1 (the unfinished farm is not a gatherable)", got)
	}
	if got := decide(t, config, gatherNodeView(0), "nodes")["nodes"]; got != 1 {
		t.Errorf("nodes with an empty farm = %v, want 1", got)
	}
}

func TestGatherVerbReachesABuildingOrAResourceByRank(t *testing.T) {
	config := rule(`{"action":"select","actors":{"kind":"units"},"pick":{"kind":"gatherables"},"match":{"score":"0 - distance(actor, target)"},"issue":[{"gather":{}}]}`)
	for farm_space, want := range map[int]int64{1: 30, 5: 7} {
		view := gatherNodeView(100)
		view.buildings[0].Location = ZoneRef{Space: Coordinate{X: farm_space}}
		if orders := selectOrders(t, config, view); len(orders) != 1 || orders[0].OrderType != kindGather || orders[0].TargetID != want {
			t.Errorf("farm at %d: orders %+v, want one gather order on node %d", farm_space, orders, want)
		}
	}
}

func attackTargetsView(building_health float64) selectTestView {
	soldier := UnitView{InternalID: 1, Kind: UnitMilitary, Type: UnitTypeInfantry, CanAttack: true}
	enemy_unit := UnitView{InternalID: 5, PlayerID: 1, Kind: UnitMilitary, Type: UnitTypeInfantry, Health: 20, MaxHealth: 20, Location: ZoneRef{Space: Coordinate{X: 2}}}
	enemy_building := BuildingView{InternalID: 5, PlayerID: 1, BuildingID: 22, Health: building_health, MaxHealth: 100, Visible: true, Location: ZoneRef{Space: Coordinate{X: 3}}}
	return selectTestView{
		loopTestView: loopTestView{units: []UnitView{soldier}, enemy_units: []UnitView{enemy_unit}, enemy_buildings: []BuildingView{enemy_building}},
		idle:         []UnitView{soldier}, turn: 1,
	}
}

func TestAttackablesAreEnemyUnitsAndBuildingsWithDistinctIdentities(t *testing.T) {
	config := rule(`{"action":"select","pick":{"kind":"attackables"},"into":{"set":"targets"}},
		{"action":"set_var","name":"saved","value":{"count":{"kind":"attackables","where":{"in_set":"targets"}}}},
		{"action":"set_var","name":"buildings","value":{"sum":{"kind":"attackables","of":"var(item.is_building)"}}}`)
	if got := decide(t, config, attackTargetsView(50), "saved", "buildings"); got["saved"] != 2 || got["buildings"] != 1 {
		t.Errorf("saved targets and buildings = %v, want 2 and 1 (a unit and a building share id 5)", got)
	}
}

func TestAttackablesCarryATargetCategoryUsableByName(t *testing.T) {
	config := rule(`{"action":"set_var","name":"sum","value":{"sum":{"kind":"attackables","of":"var(item.target_category)"}}},
		{"action":"set_var","name":"military","value":{"count":{"kind":"attackables","where":{"in":{"target_category":["military"]}}}}}`)
	if got := decide(t, config, attackTargetsView(50), "sum", "military"); got["sum"] != 5 || got["military"] != 1 {
		t.Errorf("category sum and military count = %v, want 5 (military 2 + building 3) and 1", got)
	}
}

func TestAttackablesNeverIncludeYourOwnUnitsOrBuildings(t *testing.T) {
	config := rule(`{"action":"set_var","name":"all","value":{"count":{"kind":"attackables","where":{"owner":"any"}}}}`)
	if got := decide(t, config, attackTargetsView(50), "all")["all"]; got != 2 {
		t.Errorf("attackables with owner any = %v, want 2 (only the enemy unit and building)", got)
	}
	if loadError(rule(`{"action":"set_var","name":"all","value":{"count":{"kind":"attackables","where":{"owner":"own"}}}}`)) == nil {
		t.Error("owner own accepted for attackables")
	}
}

func TestAttackVerbPicksTheWeakestTargetAcrossKinds(t *testing.T) {
	config := rule(`{"action":"select","actors":{"kind":"units","where":{"unit_types":["military"]}},"pick":{"kind":"attackables"},"match":{"score":"0 - var(target.health)"},"issue":[{"attack":{}}]}`)
	for building_health, want := range map[float64]uint8{50: kindAttackUnit, 10: kindAttackBuilding} {
		if orders := selectOrders(t, config, attackTargetsView(building_health)); len(orders) != 1 || orders[0].OrderType != want || orders[0].TargetID != 5 {
			t.Errorf("building health %v: orders %+v, want one order of kind %d on id 5", building_health, orders, want)
		}
	}
}

func TestSelectExpressionFailuresExcludeCandidates(t *testing.T) {
	worker := villager(1, Coordinate{})
	view := selectTestView{loopTestView: loopTestView{units: []UnitView{worker}, spaces: lineOfSpaces(0, 2)}, idle: []UnitView{worker}}
	for _, fields := range []string{
		`"rank":"0 - 1 / var(target.gold_income)","limit":1`,
		`"let":{"broken":"1 / var(target.gold_income)"},"where":{"if":"1 - var(target.broken)"}`,
		`"let":{"broken":"1 / var(target.gold_income)"},"where":{"in":{"broken":[0,0.5]}}`,
		`"rank":{"sum":{"kind":"spaces","where":{"if":"1 - var(item.distance_to_target)"},"of":"1 / var(item.gold_income)"}},"limit":1`,
	} {
		config := rule(`{"action":"select","pick":{"kind":"spaces",` + fields + `},"into":{"set":"chosen"}},
			{"action":"select","pick":{"kind":"spaces","where":{"in_set":"chosen"}},"issue":[{"move":{}}]}`)
		if orders := selectOrders(t, config, view); len(orders) != 1 || orders[0].TargetID != spaceCode(at(2).Space) {
			t.Errorf("%s: got %+v, want only the space whose expression succeeds", fields, orders)
		}
	}
}

func TestSelectLetMemoizationCachesImpure(t *testing.T) {
	worker := villager(1, Coordinate{})
	view := selectTestView{loopTestView: loopTestView{units: []UnitView{worker}, spaces: lineOfSpaces(1)}, idle: []UnitView{worker}}
	rng := rand.New(rand.NewSource(42))
	config := rule(`{"action":"select","pick":{"kind":"spaces","let":{"r":"random()"},"where":{"if":"var(target.r)"},"rank":"var(target.r)"},"into":{"set":"chosen"}}`)
	var raw map[string]any
	_ = json.Unmarshal([]byte(config), &raw)
	ParseModelWithDiagnostics(raw, rng, io.Discard).DecideOrders(view)
	expectedRng := rand.New(rand.NewSource(42))
	expectedRng.Float64()
	if rng.Int63() != expectedRng.Int63() {
		t.Fatalf("impure let was re-rolled; expected 1 random call, got state mismatch")
	}
}
