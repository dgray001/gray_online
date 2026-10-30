package unit

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"slices"
	"strings"
	"testing"

	. "github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/util"
)

// A game view with just what loops read; any other View method panics, which flags a loop reaching for something it should not
type loopTestView struct {
	View
	me                         int
	players                    []int
	units, enemy_units         []UnitView
	buildings, enemy_buildings []BuildingView
	spaces                     []SpaceInfo
	regions                    []RegionInfo
	mercenaries                []Producible
	zones                      []ZoneInfo
	resources                  map[ResourceCategory][]ResourceView
}

func (v loopTestView) Revision() uint64              { return 0 }
func (v loopTestView) OrderQueue() []OrderEntry      { return nil }
func (v loopTestView) Units() []UnitView             { return v.units }
func (v loopTestView) VisibleEnemyUnits() []UnitView { return v.enemy_units }
func (v loopTestView) AllUnits() []UnitView          { return slices.Concat(v.units, v.enemy_units) }
func (v loopTestView) AllBuildings() []BuildingView {
	return slices.Concat(v.buildings, v.enemy_buildings)
}
func (v loopTestView) Buildings() []BuildingView                        { return v.buildings }
func (v loopTestView) KnownEnemyBuildings() []BuildingView              { return v.enemy_buildings }
func (v loopTestView) AllSpaces() []SpaceInfo                           { return v.spaces }
func (v loopTestView) AllRegions() []RegionInfo                         { return v.regions }
func (v loopTestView) AvailableMercenaries() []Producible               { return v.mercenaries }
func (v loopTestView) AllZones() []ZoneInfo                             { return v.zones }
func (v loopTestView) PlayerIDs() []int                                 { return v.players }
func (v loopTestView) PlayerID() int                                    { return v.me }
func (v loopTestView) PlayerScore(player_id int) int                    { return 10 * player_id }
func (v loopTestView) PlayerEliminated(player_id int) bool              { return false }
func (v loopTestView) KnownResources(c ResourceCategory) []ResourceView { return v.resources[c] }

func (v loopTestView) Logger() *util.Logger { return nil }
func (v loopTestView) Nickname() string     { return "loop-test" }

// The game counters every expression can read; none of these tests depend on them
func (v loopTestView) Population() (int, int)            { return 0, 0 }
func (v loopTestView) MaxPopulation() int                { return 100 }
func (v loopTestView) TurnNumber() int                   { return 1 }
func (v loopTestView) MapSize() int                      { return 4 }
func (v loopTestView) NumPlayers() int                   { return len(v.players) }
func (v loopTestView) Resource(ResourceCategory) float64 { return 0 }

// Filters only by the space owner, which is all these tests ask of a space condition
func (v loopTestView) MatchingSpaces(condition SpaceCondition) []Coordinate {
	matches := make([]Coordinate, 0)
	for _, s := range v.spaces {
		if condition.Owner == "own" && s.Owner != v.me {
			continue
		}
		if condition.Owner == "enemy" && (s.Owner < 0 || s.Owner == v.me) {
			continue
		}
		matches = append(matches, s.Space)
	}
	return matches
}

func (v loopTestView) SpaceDistance(a Coordinate, b Coordinate) int {
	dx, dy := b.X-a.X, b.Y-a.Y
	return max(max(dx, -dx), max(dy, -dy), max(dx+dy, -dx-dy))
}

func (v loopTestView) SpaceNeighbors(space Coordinate) []Coordinate {
	return nil
}

// The error a config fails to load with, or nil
func loadError(config string) error {
	var raw map[string]any
	if err := json.Unmarshal([]byte(config), &raw); err != nil {
		return err
	}
	if _, ok := ParseModelWithDiagnostics(raw, nil, io.Discard).(*RulesModel); !ok {
		return fmt.Errorf("invalid AI config")
	}
	return nil
}

func rule(then string) string {
	return `{"rules": [{"when": {"always": {}}, "then": [` + then + `]}]}`
}

// Players 0 (us, home at 0,0), 1 (a building at 3,0 and a unit) and 2 (only a unit)
func loopFixture() loopTestView {
	return loopTestView{
		me: 0, players: []int{0, 1, 2},
		units:           []UnitView{{InternalID: 1, PlayerID: 0, Health: 4}, {InternalID: 2, PlayerID: 0, Health: 6}, {InternalID: 3, PlayerID: 0, Health: 9}},
		enemy_units:     []UnitView{{InternalID: 11, PlayerID: 1}, {InternalID: 12, PlayerID: 2}},
		buildings:       []BuildingView{{InternalID: 20, PlayerID: 0, BuildingID: 1}},
		enemy_buildings: []BuildingView{{InternalID: 21, PlayerID: 1, BuildingID: 22, Location: ZoneRef{Space: Coordinate{X: 3}}}},
	}
}

func TestForEachLimitFailureAndSuccessfulLimits(t *testing.T) {
	for limit, want := range map[string]float64{"": 3, `,"max":0`: 3, `,"max":2`: 2, `,"max":"1 / (gameVar(turn) - gameVar(turn))"`: 0} {
		config := rule(`{"action":"set_var","name":"visited","value":0},
			{"action":"for_each","source":"units","as":"u"` + limit + `,"do":[
			{"action":"set_var","name":"visited","value":"var(visited) + 1"}]}`)
		if got := decide(t, config, loopFixture(), "visited")["visited"]; got != want {
			t.Errorf("limit %s: visited %v, want %v", limit, got, want)
		}
	}
}

func TestForEachSelectSupportsSelectorComposition(t *testing.T) {
	base := loopFixture()
	view := selectTestView{loopTestView: base, idle: base.units, turn: 1}
	config := rule(`{"action":"select","pick":{"kind":"units","where":{"in":{"uid":[1,2]}}},"into":{"set":"kept"}},
		{"action":"for_each","select":{"kind":"units","where":{"in_set":"kept"},"rank":"var(u.health)","skip":1,"limit":1,
		"let":{"double_health":"2 * var(u.health)"}},"as":"u","do":[{"action":"set_var","name":"result","value":"var(u.uid) + var(u.double_health)"}]}`)
	if got := decide(t, config, view, "result")["result"]; got != 9 {
		t.Fatalf("selector loop result = %v, want unit 1 after filtering, ranking and skip", got)
	}
}

func TestSelectorRankArrayBreaksTiesInOrder(t *testing.T) {
	for rank, wantID := range map[string]float64{
		`["1","var(target.health)"]`:                                           3,
		`["0 - var(target.health)","var(target.uid)"]`:                         1,
		`["var(target.health) - var(target.health)","0 - var(target.health)"]`: 1,
		`"var(target.health)"`:                                                 3,
	} {
		base := loopFixture()
		view := selectTestView{loopTestView: base, idle: base.units, turn: 1}
		config := rule(`{"action":"select","pick":{"kind":"units","rank":` + rank + `,"limit":1},"into":{"set":"top"}},
			{"action":"for_each","select":{"kind":"units","where":{"in_set":"top"}},"as":"u","do":[{"action":"set_var","name":"result","value":"var(u.uid)"}]}`)
		if got := decide(t, config, view, "result")["result"]; got != wantID {
			t.Errorf("rank %s: picked unit %v, want %v", rank, got, wantID)
		}
	}
}

func TestSelectorDrawsOnlyWithinTies(t *testing.T) {
	for name, c := range map[string]struct {
		rank string
		pool int
	}{
		"unique ranks":     {`"var(target.health)"`, 0},
		"three tied":       {`"1"`, 3},
		"three tied lists": {`["1","2 - 1"]`, 3},
		"second key ties":  {`["var(target.health)","1"]`, 0},
	} {
		var raw map[string]any
		config := rule(`{"action":"select","pick":{"kind":"units","rank":` + c.rank + `,"limit":1},"into":{"set":"picked"}}`)
		if err := json.Unmarshal([]byte(config), &raw); err != nil {
			t.Fatal(err)
		}
		used, expected := rand.New(rand.NewSource(7)), rand.New(rand.NewSource(7))
		base := loopFixture()
		ParseModelWithDiagnostics(raw, used, io.Discard).DecideOrders(selectTestView{loopTestView: base, idle: base.units, turn: 1})
		if c.pool > 0 {
			expected.Intn(c.pool)
		}
		if used.Int63() != expected.Int63() {
			t.Errorf("%s: generator state differs from %d draw(s) of Intn(%d)", name, min(1, c.pool), c.pool)
		}
	}
}

func TestForEachSelectRejectsReservedItemNames(t *testing.T) {
	for _, name := range []string{"actor", "target", "item", "anchor", "place"} {
		config := rule(fmt.Sprintf(`{"action":"for_each","select":{"kind":"units"},"as":%q,"do":[]}`, name))
		if loadError(config) == nil {
			t.Errorf("selector loop named %q accepted", name)
		}
	}
}

func TestForEachSelectAggregatesSeeOrdersFromTheBody(t *testing.T) {
	for name, loop := range map[string]string{"source": `"source":"units"`, "select": `"select":{"kind":"units","limit":1}`} {
		view := productionView()
		view.visibleEnemyUnits[0].PlayerID = 1
		model := actionsModel(t, `{"action":"for_each",`+loop+`,"as":"u","do":[
			{"action":"select","actors":{"kind":"buildings","where":{"in":{"idle":[1]},"can_create":2},"limit":1},"issue":[{"create":{"unit_id":2}}]},
			{"action":"set_var","name":"idle","value":{"count":{"kind":"buildings","where":{"in":{"idle":[1]}}}}}]}`).(*RulesModel)
		model.DecideOrders(view)
		if idle, _ := model.Var("idle"); idle != 1 {
			t.Errorf("%s loop: idle buildings after queuing = %v, want 1", name, idle)
		}
	}
}

func TestForEachPlayersFoundByBuildingOrUnit(t *testing.T) {
	config := func(has_eco int) string {
		return rule(fmt.Sprintf(`{"action": "set_var", "name": "has_eco", "value": %d},
			{"action": "set_var", "name": "found", "value": 0, "persist": true},
			{"action": "for_each", "source": "players", "as": "p", "where": {"owner": "enemy"}, "do": [
				{"action": "if", "when": {"value_at_least": {"value": "var(has_eco) * var(p.buildings_known) + (1 - var(has_eco)) * var(p.units_visible)", "amount": 1}},
				 "then": [{"action": "set_var", "name": "found", "value": "var(found) + 1", "persist": true}]}]}`, has_eco))
	}
	for has_eco, want := range map[int]float64{1: 1, 0: 2} {
		if got := decide(t, config(has_eco), loopFixture(), "found")["found"]; got != want {
			t.Errorf("has_eco %d: found %v players, want %v", has_eco, got, want)
		}
	}
}

func TestForEachNestedWithExpressionFilter(t *testing.T) {
	view := loopFixture()
	view.enemy_buildings = append(view.enemy_buildings, BuildingView{InternalID: 22, PlayerID: 2}, BuildingView{InternalID: 23, PlayerID: 2})
	config := rule(`{"action": "set_var", "name": "pairs", "value": 0, "persist": true},
		{"action": "for_each", "source": "players", "as": "p", "where": {"owner": "enemy"}, "do": [
			{"action": "for_each", "source": "buildings", "as": "b", "where": {"owner": "enemy", "player_id": "var(p.id)"}, "do": [
				{"action": "set_var", "name": "pairs", "value": "var(pairs) + var(p.id)", "persist": true}]}]}`)
	// player 1 owns 1 building and player 2 owns 2, so 1 * 1 + 2 * 2
	if got := decide(t, config, view, "pairs")["pairs"]; got != 5 {
		t.Errorf("pairs = %v, want 5", got)
	}
}

func TestForEachEnemySpacesInFixedOrder(t *testing.T) {
	view := loopFixture()
	three, five := 3, 5
	view.spaces = []SpaceInfo{
		{Space: Coordinate{X: 2}, Owner: 1, UnitCount: &five},
		{Space: Coordinate{X: 1}, Owner: 1, UnitCount: &three},
		{Space: Coordinate{X: 5}, Owner: 1, UnitCount: &five},
		{Space: Coordinate{Y: 1}, Owner: 0},
	}
	config := rule(`{"action": "set_var", "name": "unidentified", "value": 0, "persist": true},
		{"action": "for_each", "source": "spaces", "as": "s", "where": {"owner": "enemy"}, "do": [
			{"action": "set_var", "name": "unidentified", "value": "var(unidentified) + var(s.unidentified_count)", "persist": true}]},
		{"action": "for_each", "source": "spaces", "as": "s", "max": 1, "do": [
			{"action": "set_var", "name": "first_x", "value": "var(s.x)", "persist": true}]}`)
	vars := decide(t, config, view, "unidentified", "first_x")
	if vars["unidentified"] != 13 {
		t.Errorf("unidentified units in enemy spaces = %v, want 13", vars["unidentified"])
	}
	if vars["first_x"] != 0 {
		t.Errorf("first space visited has x %v, want the lowest coordinate 0", vars["first_x"])
	}
}

func TestForEachSpaceIncomeAndRegionBonus(t *testing.T) {
	view := loopFixture()
	view.me = 0
	view.spaces = []SpaceInfo{{Owner: 0, GoldIncome: 3}, {Space: Coordinate{X: 1}, Owner: 1, GoldIncome: 7}}
	view.regions = []RegionInfo{{Owner: 0, GoldBonus: 5}, {Owner: 1, GoldBonus: 11}}
	config := rule(`{"action":"for_each","source":"spaces","as":"s","do":[{"action":"if","when":{"value_equals":{"value":"var(s.owner)","amount":0}},"then":[
		{"action":"set_var","name":"income","value":"var(income) + var(s.gold_income)","persist":true}]}]},
		{"action":"for_each","source":"regions","as":"r","do":[{"action":"if","when":{"value_equals":{"value":"var(r.owner)","amount":0}},"then":[
			{"action":"set_var","name":"income","value":"var(income) + var(r.gold_bonus)","persist":true}]}]}`)
	if got := decide(t, config, view, "income")["income"]; got != 8 {
		t.Errorf("income = %v, want owned space 3 + region 5", got)
	}
}

func TestForEachMercenaryCost(t *testing.T) {
	view := loopFixture()
	view.mercenaries = []Producible{{ID: 101, Cost: Cost{Gold: 6}, StaminaCost: 3}}
	config := rule(`{"action":"for_each","source":"mercenaries","as":"m","do":[
		{"action":"set_var","name":"merc_id","value":"var(m.id)","persist":true},
		{"action":"set_var","name":"merc_stamina","value":"var(m.stamina_cost)","persist":true},
		{"action":"set_var","name":"merc_cost","value":"var(m.cost_gold)","persist":true}]}`)
	vars := decide(t, config, view, "merc_id", "merc_cost", "merc_stamina")
	if vars["merc_id"] != 101 || vars["merc_cost"] != 6 || vars["merc_stamina"] != 3 {
		t.Errorf("mercenary id %v cost %v stamina %v, want 101, 6 and 3", vars["merc_id"], vars["merc_cost"], vars["merc_stamina"])
	}
}

func TestForEachUnitsMaxResourcesAndZones(t *testing.T) {
	view := loopFixture()
	view.resources = map[ResourceCategory][]ResourceView{ResourceFood: {{AmountLeft: 30}, {AmountLeft: 12}}, ResourceWood: {{AmountLeft: 7}}}
	view.zones = []ZoneInfo{{BuildingPlayer: 1}, {BuildingPlayer: -1, HasResource: true}, {BuildingPlayer: 0}}
	config := rule(`{"action": "for_each", "source": "units", "as": "u", "max": 2, "do": [
			{"action": "set_var", "name": "health", "value": "var(health) + var(u.health)", "persist": true}]},
		{"action": "for_each", "source": "resources", "as": "r", "where": {"category": "food"}, "do": [
			{"action": "set_var", "name": "food_left", "value": "var(food_left) + var(r.amount_left)", "persist": true}]},
		{"action": "for_each", "source": "zones", "as": "z", "where": {"building": true, "owner": "enemy"}, "do": [
			{"action": "set_var", "name": "enemy_zones", "value": "var(enemy_zones) + 1", "persist": true}]}`)
	vars := decide(t, config, view, "health", "food_left", "enemy_zones")
	if vars["health"] != 10 || vars["food_left"] != 42 || vars["enemy_zones"] != 1 {
		t.Errorf("health %v, food %v, enemy zones %v; want 10, 42, 1", vars["health"], vars["food_left"], vars["enemy_zones"])
	}
}

func TestLoopConfigErrors(t *testing.T) {
	loop := func(source string, where string, body string) string {
		return rule(`{"action": "for_each", "source": "` + source + `", "as": "b", ` + where + `"do": [` + body + `]}`)
	}
	set := func(value string) string {
		return `{"action": "set_var", "name": "x", "value": "` + value + `"}`
	}
	cases := map[string]struct{ config, want string }{
		"unknown source":         {loop("ships", "", set("1")), "source"},
		"unknown field":          {loop("buildings", "", set("var(b.nope)")), "no field"},
		"read outside a loop":    {rule(set("var(b.x)")), "outside a for_each"},
		"unknown where key":      {loop("units", `"where": {"colour": 1}, `, set("1")), "unknown where key"},
		"bad owner":              {loop("units", `"where": {"owner": "friend"}, `, set("1")), "unknown owner"},
		"no body":                {rule(`{"action": "for_each", "source": "units", "as": "u"}`), "do"},
		"item of another source": {loop("players", "", set("var(b.health)")), "no field"},
		"source and select":      {rule(`{"action":"for_each","source":"units","select":{"kind":"units"},"as":"u","do":[]}`), "cannot be combined"},
		"select and where":       {rule(`{"action":"for_each","select":{"kind":"units"},"where":{},"as":"u","do":[]}`), "cannot be combined"},
	}
	for name, c := range cases {
		err := loadError(c.config)
		if err == nil {
			t.Errorf("%s: malformed config accepted (%s)", name, c.want)
		}
	}
}

func TestLoopNestingAndStepLimits(t *testing.T) {
	body := `{"action": "set_var", "name": "n", "value": "var(n) + 1", "persist": true}`
	for depth := 0; depth < maxNesting; depth++ {
		body = `{"action": "for_each", "source": "players", "as": "p` + fmt.Sprint(depth) + `", "do": [` + body + `]}`
	}
	if err := loadError(rule(body)); err != nil {
		t.Errorf("loops nested %d deep: %v", maxNesting, err)
	}
	if err := loadError(rule(`{"action": "for_each", "source": "players", "as": "x", "do": [` + body + `]}`)); err == nil {
		t.Errorf("loops nested %d deep: error %v, want a nesting error", maxNesting+1, err)
	}
	view := loopFixture()
	for i := 0; i < 40; i++ {
		view.players = append(view.players, 3+i)
	}
	if got := decide(t, rule(body), view, "n")["n"]; got > maxLoopSteps || got == 0 {
		t.Errorf("loop bodies ran %v times, want at most %d", got, maxLoopSteps)
	}
}

func TestNestedLoopReusingANameRestoresTheOuterItem(t *testing.T) {
	config := rule(`{"action": "for_each", "source": "players", "as": "x", "where": {"owner": "enemy"}, "do": [
		{"action": "for_each", "source": "buildings", "as": "x", "where": {"owner": "own"}, "do": []},
		{"action": "set_var", "name": "after", "value": "var(after) + var(x.id)", "persist": true}]}`)
	// the inner x shadowed the player inside its loop only, so after it the outer x is player 1 then player 2
	if got := decide(t, config, loopFixture(), "after")["after"]; got != 3 {
		t.Errorf("after = %v, want 3", got)
	}
}

func TestIfElse(t *testing.T) {
	config := rule(`{"action": "if", "when": {"value_at_least": {"value": "var(missing)", "amount": 1}},
		"then": [{"action": "set_var", "name": "branch", "value": 1, "persist": true}],
		"else": [{"action": "set_var", "name": "branch", "value": 2, "persist": true}]}`)
	if got := decide(t, config, loopFixture(), "branch")["branch"]; got != 2 {
		t.Errorf("branch = %v, want the else branch 2", got)
	}
}

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
	model, ok := ParseModelWithDiagnostics(raw, rand.New(rand.NewSource(1)), io.Discard).(*RulesModel)
	if !ok {
		t.Fatalf("config did not load: %s", config)
	}
	model.DecideOrders(decisionView{view})
	values := make(map[string]float64, len(names))
	for _, name := range names {
		values[name], _ = model.Var(name)
	}
	return values
}

func TestLoopItemFields(t *testing.T) {
	count := 2
	view := loopTestView{players: []int{0},
		units:     []UnitView{{InternalID: 7, UnitID: 11, Type: UnitTypeInfantry, Kind: UnitMilitary, Health: 4, MaxHealth: 8, CurrentStamina: 3, Location: ZoneRef{Space: Coordinate{X: 1}}}},
		buildings: []BuildingView{{InternalID: 9, BuildingID: 1, UnderConstruction: true, Health: 30, MaxHealth: 50}},
		spaces:    []SpaceInfo{{Space: Coordinate{X: 1}, Vision: 3, UnitCount: &count}},
		resources: map[ResourceCategory][]ResourceView{ResourceFood: {{InternalID: 5, AmountLeft: 12}}},
		zones:     []ZoneInfo{{HasResource: true}},
	}
	place := "x y zone_x zone_y"
	for source, c := range map[string]struct {
		fields string
		want   float64
	}{
		"players":   {"id is_me score units_visible buildings_known", 3},
		"spaces":    {"vision owner unit_count unidentified_count " + place, 7},
		"units":     {"uid player unit_id unit_type is_military is_garrisoned health max_health stamina " + place, 37},
		"buildings": {"uid player building_id is_under_construction health max_health " + place, 1000000091},
		"resources": {"uid category amount_left " + place, 2000000017},
		"zones":     {"has_resource building_player " + place, 1},
	} {
		t.Run(source, func(t *testing.T) {
			var terms []string
			for _, field := range strings.Fields(c.fields) {
				terms = append(terms, "var(item."+field+")")
			}
			config := rule(fmt.Sprintf(`{"action":"for_each","source":%q,"as":"item","do":[{"action":"set_var","name":"sum","value":%q}]}`, source, strings.Join(terms, " + ")))
			if got := decide(t, config, view, "sum")["sum"]; got != c.want {
				t.Fatalf("field sum=%v, want %v", got, c.want)
			}
			bad := rule(fmt.Sprintf(`{"action":"for_each","source":%q,"as":"item","do":[{"action":"set_var","name":"bad","value":"var(item.no_such_field)"}]}`, source))
			if loadError(bad) == nil {
				t.Fatal("unknown item field accepted")
			}
		})
	}
}

func TestLoopItemRangeStaminaAndVisibilityFields(t *testing.T) {
	view := loopTestView{
		units:     []UnitView{{InternalID: 7, MaxStamina: 6, AttackRange: AttackRangeAdjacent}},
		buildings: []BuildingView{{InternalID: 9, AttackRange: AttackRangeZone, MaxStamina: 4, Visible: true}},
	}
	config := rule(`{"action":"for_each","source":"units","as":"u","do":[{"action":"set_var","name":"unit_sum","value":"var(u.max_stamina) * 100 + var(u.attack_range) * 10 + var(u.attack_reach)"}]},
		{"action":"for_each","source":"buildings","as":"b","do":[{"action":"set_var","name":"building_sum","value":"var(b.max_stamina) * 100 + var(b.attack_range) * 10 + var(b.attack_reach) + var(b.is_visible) * 1000"}]}`)
	vars := decide(t, config, view, "unit_sum", "building_sum")
	if vars["unit_sum"] != 661 || vars["building_sum"] != 1409 {
		t.Errorf("unit sum %v, building sum %v; want 661 and 1409", vars["unit_sum"], vars["building_sum"])
	}
}

func TestLoopItemBehaviorAndGarrisonFields(t *testing.T) {
	view := loopTestView{
		units:     []UnitView{{InternalID: 7, Stance: UnitStanceAggressive, AttackBack: true}},
		buildings: []BuildingView{{InternalID: 9, GarrisonCount: 2, GarrisonCapacity: 5, ResourcesLeft: 7, Renewing: true, CanAttack: true}},
	}
	config := rule(`{"action":"for_each","source":"units","as":"u","do":[{"action":"set_var","name":"unit_sum","value":"var(u.stance) + var(u.attack_back) + var(u.interrupt_current)"}]},
		{"action":"for_each","source":"buildings","as":"b","do":[{"action":"set_var","name":"building_sum","value":"var(b.garrison_count) + var(b.garrison_capacity) + var(b.resources_left) + var(b.is_renewing) + var(b.can_attack)"}]}`)
	vars := decide(t, config, view, "unit_sum", "building_sum")
	if vars["unit_sum"] != 3 || vars["building_sum"] != 16 {
		t.Errorf("unit sum %v, building sum %v; want 3 and 16", vars["unit_sum"], vars["building_sum"])
	}
}
