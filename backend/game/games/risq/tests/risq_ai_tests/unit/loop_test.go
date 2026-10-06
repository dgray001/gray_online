package unit

import (
	"encoding/json"
	"fmt"
	. "github.com/dgray001/gray_online/game/games/risq/ai"
	"testing"
)

// A game view with just what loops read; any other View method panics, which flags a loop reaching for something it should not
type loopTestView struct {
	View
	me                         int
	players                    []int
	units, enemy_units         []UnitView
	buildings, enemy_buildings []BuildingView
	spaces                     []SpaceInfo
	zones                      []ZoneInfo
	resources                  map[ResourceCategory][]ResourceView
}

func (v loopTestView) Units() []UnitView                                { return v.units }
func (v loopTestView) VisibleEnemyUnits() []UnitView                    { return v.enemy_units }
func (v loopTestView) Buildings() []BuildingView                        { return v.buildings }
func (v loopTestView) KnownEnemyBuildings() []BuildingView              { return v.enemy_buildings }
func (v loopTestView) AllSpaces() []SpaceInfo                           { return v.spaces }
func (v loopTestView) AllZones() []ZoneInfo                             { return v.zones }
func (v loopTestView) PlayerIDs() []int                                 { return v.players }
func (v loopTestView) PlayerID() int                                    { return v.me }
func (v loopTestView) PlayerScore(player_id int) int                    { return 10 * player_id }
func (v loopTestView) KnownResources(c ResourceCategory) []ResourceView { return v.resources[c] }

func (v loopTestView) Nickname() string { return "loop-test" }

// The game counters every expression can read; none of these tests depend on them
func (v loopTestView) Population() (int, int)            { return 0, 0 }
func (v loopTestView) TurnNumber() int                   { return 1 }
func (v loopTestView) MapSize() int                      { return 4 }
func (v loopTestView) NumPlayers() int                   { return len(v.players) }
func (v loopTestView) EnemiesFound() int                 { return 0 }
func (v loopTestView) OwnedSpaces() int                  { return 0 }
func (v loopTestView) Score() int                        { return 0 }
func (v loopTestView) BestEnemyScore() int               { return 0 }
func (v loopTestView) Resource(ResourceCategory) float64 { return 0 }

// Filters only by the space owner, which is all these tests ask of a space condition
func (v loopTestView) MatchingSpaces(condition SpaceCondition) []Coordinate {
	matches := make([]Coordinate, 0)
	for _, s := range v.spaces {
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

// The error a config fails to load with, or nil
func loadError(config string) error {
	var raw map[string]any
	if err := json.Unmarshal([]byte(config), &raw); err != nil {
		return err
	}
	if _, ok := ParseModel(raw, nil).(*RulesModel); !ok {
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

func TestForEachNearestEnemyBuilding(t *testing.T) {
	view := loopFixture()
	view.enemy_buildings = []BuildingView{
		{InternalID: 31, PlayerID: 1, Location: ZoneRef{Space: Coordinate{X: 3}}},
		{InternalID: 32, PlayerID: 1, Location: ZoneRef{Space: Coordinate{X: 1}}},
		{InternalID: 33, PlayerID: 2, Location: ZoneRef{Space: Coordinate{X: 2}}},
	}
	config := rule(`{"action": "set_var", "name": "best_d", "value": 1000, "persist": true},
		{"action": "for_each", "source": "buildings", "as": "b", "where": {"owner": "enemy"}, "do": [
			{"action": "if", "when": {"value_at_most": {"value": "var(b.distance_home)", "amount": "var(best_d) - 1"}},
			 "then": [{"action": "set_var", "name": "best_d", "value": "var(b.distance_home)", "persist": true},
			          {"action": "set_var", "name": "best_x", "value": "var(b.x)", "persist": true}]}]}`)
	vars := decide(t, config, view, "best_d", "best_x")
	if vars["best_d"] != 1 || vars["best_x"] != 1 {
		t.Errorf("nearest enemy building: distance %v at x %v, want 1 at 1", vars["best_d"], vars["best_x"])
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

func TestForEachSpacesNearHomeAndInFixedOrder(t *testing.T) {
	view := loopFixture()
	three, five := 3, 5
	view.spaces = []SpaceInfo{
		{Space: Coordinate{X: 2}, Owner: 1, UnitCount: &five},
		{Space: Coordinate{X: 1}, Owner: 1, UnitCount: &three},
		{Space: Coordinate{X: 5}, Owner: 1, UnitCount: &five},
		{Space: Coordinate{Y: 1}, Owner: 0},
	}
	config := rule(`{"action": "set_var", "name": "unidentified", "value": 0, "persist": true},
		{"action": "for_each", "source": "spaces", "as": "s", "where": {"owner": "enemy", "within": 2}, "do": [
			{"action": "set_var", "name": "unidentified", "value": "var(unidentified) + var(s.unidentified_count)", "persist": true}]},
		{"action": "for_each", "source": "spaces", "as": "s", "max": 1, "do": [
			{"action": "set_var", "name": "first_x", "value": "var(s.x)", "persist": true}]}`)
	vars := decide(t, config, view, "unidentified", "first_x")
	if vars["unidentified"] != 8 {
		t.Errorf("unidentified units within 2 of home = %v, want 8", vars["unidentified"])
	}
	if vars["first_x"] != 0 {
		t.Errorf("first space visited has x %v, want the lowest coordinate 0", vars["first_x"])
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
