package unit

import (
	"encoding/json"
	"io"
	"math/rand"
	"testing"

	. "github.com/dgray001/gray_online/game/games/risq/ai"
)

type noHomeSelectView struct{ selectTestView }

func (v noHomeSelectView) Buildings() []BuildingView { panic("select attempted to locate home") }

func TestSelectSourcesDoNotLocateHome(t *testing.T) {
	view := noHomeSelectView{selectTestView{loopTestView: loopTestView{
		units: []UnitView{villager(1, Coordinate{})}, buildings: []BuildingView{{InternalID: 1}},
		spaces: lineOfSpaces(0), zones: []ZoneInfo{{Location: at(1)}},
		resources: map[ResourceCategory][]ResourceView{ResourceWood: {{InternalID: 1, Location: at(1), AmountLeft: 10}}},
	}}}
	for _, kind := range []string{"units", "buildings", "spaces", "zones", "resources"} {
		selectOrders(t, rule(`{"action":"select","pick":{"kind":"`+kind+`"},"into":{"set":"chosen"}}`), view)
	}
}

func seededModel(t *testing.T, config string, seed int64) *RulesModel {
	t.Helper()
	var raw map[string]any
	if err := json.Unmarshal([]byte(config), &raw); err != nil {
		t.Fatal(err)
	}
	model, ok := ParseModelWithDiagnostics(raw, rand.New(rand.NewSource(seed)), io.Discard).(*RulesModel)
	if !ok {
		t.Fatalf("config did not load: %s", config)
	}
	return model
}

func chosenSpaces(t *testing.T, config string, view View) map[int64]int {
	t.Helper()
	chosen := map[int64]int{}
	for seed := int64(1); seed <= 40; seed++ {
		orders := seededModel(t, config, seed).DecideOrders(view).Orders
		if len(orders) != 1 {
			t.Fatalf("seed %d: orders = %+v, want one", seed, orders)
		}
		chosen[orders[0].TargetID]++
	}
	return chosen
}

func tieView() selectTestView {
	worker := villager(1, Coordinate{})
	return selectTestView{loopTestView: loopTestView{units: []UnitView{worker}, spaces: lineOfSpaces(make([]float64, 12)...)}, idle: []UnitView{worker}, turn: 1}
}

func TestSelectBreaksTiesRandomlyAndReproducibly(t *testing.T) {
	config := rule(`{"action":"select","pick":{"kind":"spaces","rank":"1","limit":1},"issue":[{"move":{}}]}`)
	if got := chosenSpaces(t, config, tieView()); len(got) < 4 {
		t.Errorf("tied spaces chosen across 40 seeds = %v, want a spread, not always the lowest coordinate", got)
	}
	first := seededModel(t, config, 7).DecideOrders(tieView()).Orders
	second := seededModel(t, config, 7).DecideOrders(tieView()).Orders
	if len(first) != 1 || len(second) != 1 || first[0].TargetID != second[0].TargetID {
		t.Errorf("same seed chose %+v then %+v, want identical", first, second)
	}
}

func TestSelectCoordinateTiebreakWhenAskedFor(t *testing.T) {
	config := rule(`{"action":"select","pick":{"kind":"spaces","rank":"1 - var(target.x) / 100","limit":1},"issue":[{"move":{}}]}`)
	if got := chosenSpaces(t, config, tieView()); len(got) != 1 || got[100] != 40 {
		t.Errorf("explicit coordinate tiebreak chose %v, want always x=1", got)
	}
}

func TestRandomIsCallableInExpressions(t *testing.T) {
	config := rule(`{"action":"select","pick":{"kind":"spaces","rank":"random()","limit":1},"issue":[{"move":{}}]}`)
	if got := chosenSpaces(t, config, tieView()); len(got) < 4 {
		t.Errorf("random() ranking chose %v across 40 seeds, want a spread", got)
	}
	roll := decide(t, rule(`{"action":"set_var","name":"roll","value":"randomInt(3, 5)"},{"action":"set_var","name":"unit","value":"random()"}`), tieView(), "roll", "unit")
	if roll["roll"] < 3 || roll["roll"] > 5 {
		t.Errorf("randomInt(3, 5) = %v, want a whole number from 3 to 5", roll["roll"])
	}
	if loadError(rule(`{"action":"set_var","name":"roll","value":"random() + 1"}`)) != nil {
		t.Error("random() rejected at load")
	}
}

func TestExploreRewrittenInJSON(t *testing.T) {
	near, far := villager(1, Coordinate{}), villager(2, Coordinate{X: 6})
	view := selectTestView{
		loopTestView: loopTestView{
			units: []UnitView{near, far},
			spaces: []SpaceInfo{
				{Space: Coordinate{X: 1}, Vision: 3}, {Space: Coordinate{X: 2}}, {Space: Coordinate{X: 5}}, {Space: Coordinate{X: 9}},
			},
		},
		idle: []UnitView{near, far}, turn: 1,
	}
	explore := func(to string) string {
		return rule(`{"action":"select","pick":{"kind":"spaces","where":{"if":"1 - var(target.vision)"}},
			"assign":{"capacity":1,"overflow":"none","unit_score":"0 - var(target.distance_to_actor)"},
			"issue":[{"move":{"to":"` + to + `"}}]}`)
	}
	for _, c := range []struct {
		to   string
		kind uint8
	}{{"zone", kindMoveZone}, {"space", kindMoveSpace}} {
		got := map[uint64]int64{}
		for _, o := range selectOrders(t, explore(c.to), view) {
			if o.OrderType != c.kind {
				t.Fatalf("to %s: order %+v, want kind %d", c.to, o, c.kind)
			}
			got[o.Subjects[0]] = o.TargetID
		}
		if got[1] != 200 || got[2] != 500 {
			t.Errorf("to %s: explorers went to %v, want unit 1 to x=2 and unit 2 to x=5", c.to, got)
		}
	}
}

func TestTurnsSinceSeenSupportsFogRevisits(t *testing.T) {
	worker := villager(1, Coordinate{})
	seen := []SpaceInfo{{Space: Coordinate{X: 1}, Vision: 3}, {Space: Coordinate{X: 2}, Vision: 3}}
	view := selectTestView{loopTestView: loopTestView{units: []UnitView{worker}, spaces: seen}, idle: []UnitView{worker}, turn: 1}
	config := rule(`{"action":"select","pick":{"kind":"spaces","where":{"if":["var(target.turns_since_seen) - 3","var(target.vision) * (2 - var(target.vision))"]},"limit":1},"issue":[{"move":{}}]}`)
	model := seededModel(t, config, 1)
	model.ApplyUpdate(view, "start-turn")
	view.turn = 2
	view.spaces = []SpaceInfo{{Space: Coordinate{X: 1}, Vision: 3}, {Space: Coordinate{X: 2}, Vision: 1}}
	model.ApplyUpdate(view, "start-turn")
	if orders := model.DecideOrders(view).Orders; len(orders) != 0 {
		t.Fatalf("turn 2 orders = %+v, want none: the fogged space was seen one turn ago", orders)
	}
	view.turn = 6
	model.ApplyUpdate(view, "start-turn")
	orders := model.DecideOrders(view).Orders
	if len(orders) != 1 || orders[0].TargetID != 200 {
		t.Fatalf("turn 6 orders = %+v, want a revisit of the space last seen on turn 1 (x=2)", orders)
	}
}

func TestAttackAndMoveTargetKindIsChoosable(t *testing.T) {
	worker := villager(1, Coordinate{})
	worker.CanAttack = true
	view := selectTestView{
		loopTestView: loopTestView{units: []UnitView{worker}, enemy_units: []UnitView{{InternalID: 50, PlayerID: 1, Location: ZoneRef{Space: Coordinate{X: 3}}}}},
		idle:         []UnitView{worker}, turn: 1,
	}
	for _, c := range []struct {
		to   string
		kind uint8
	}{{"", kindAttackUnit}, {"space", kindAttackSpace}, {"zone", kindAttackZone}} {
		options := `{}`
		if c.to != "" {
			options = `{"to":"` + c.to + `"}`
		}
		config := rule(`{"action":"select","pick":{"kind":"units","where":{"owner":"enemy"}},"issue":[{"attack":` + options + `}]}`)
		if orders := selectOrders(t, config, view); len(orders) != 1 || orders[0].OrderType != c.kind {
			t.Errorf("attack to %q: orders = %+v, want kind %d", c.to, orders, c.kind)
		}
	}
	if loadError(rule(`{"action":"select","pick":{"kind":"spaces"},"issue":[{"move":{"to":"sector"}}]}`)) == nil {
		t.Error("unknown move target kind accepted")
	}
}
