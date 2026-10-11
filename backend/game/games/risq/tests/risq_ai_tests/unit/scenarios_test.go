package unit

import (
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"reflect"
	"testing"
)

type namedCall struct {
	label string
	call  orderCall
}

func TestEconomyScenario(t *testing.T) {
	model := requireModel(t, economyScenario)
	view := economyView(1)
	decision := model.DecideOrders(view)
	assertScenarioCalls(t, "initial allocation", decision, view, []namedCall{
		{"research", orderCall{Kind: 14, Subject: 800, Item: 3}},
		{"foundation", orderCall{Kind: 3, Subject: 91, Item: 2, Place: ai.ZoneRef{Space: ai.Coordinate{X: 2}}, Clear: true}},
		{"queued unit", orderCall{Kind: 13, Subject: 801, Item: 2}},
		{"queue top up", orderCall{Kind: 13, Subject: 801, Item: 2}},
		{"last population slot", orderCall{Kind: 19, Item: 2, Place: ai.ZoneRef{Zone: ai.Coordinate{X: 1}}}},
		{"remaining worker", orderCall{Kind: 2, Subject: 92, Target: 401, Place: ai.ZoneRef{Space: ai.Coordinate{X: 3}}, Clear: true}},
		{"budget probe", orderCall{Kind: 1, Subject: 99, Place: ai.ZoneRef{Space: ai.Coordinate{X: 108, Y: 2503}}, Clear: true}},
	})
	if want := []resourceQuery{{Category: ai.ResourceWood, Gatherer: 92}}; !reflect.DeepEqual(view.resourceQueries, want) {
		t.Errorf("initial allocation/resource query: got %+v, want %+v", view.resourceQueries, want)
	}
	view = economyView(2)
	assertScenarioCalls(t, "refreshed budget and retained workers", model.DecideOrders(view), view, []namedCall{
		{"first worker", orderCall{Kind: 2, Subject: 91, Target: 401, Place: ai.ZoneRef{Space: ai.Coordinate{X: 3}}, Clear: true}},
		{"second worker", orderCall{Kind: 2, Subject: 92, Target: 401, Place: ai.ZoneRef{Space: ai.Coordinate{X: 3}}, Clear: true}},
		{"refreshed budget", orderCall{Kind: 1, Subject: 99, Place: ai.ZoneRef{Space: ai.Coordinate{X: 205, Y: 3009}}, Clear: true}},
	})
	view = economyView(3)
	view.buildings[0].Health, view.buildings[1].ResourcesLeft = 5, 0
	assertScenarioCalls(t, "maintenance and shared garrison capacity", model.DecideOrders(view), view, []namedCall{
		{"one repairer", orderCall{Kind: 4, Subject: 91, Target: 800, Place: ai.ZoneRef{Zone: ai.Coordinate{X: 1}}, Clear: true}},
		{"one renewer", orderCall{Kind: 5, Subject: 92, Target: 801, Place: ai.ZoneRef{Zone: ai.Coordinate{X: 1}}, Clear: true}},
		{"one garrison slot", orderCall{Kind: 10, Subject: 93, Target: 800, Place: ai.ZoneRef{Zone: ai.Coordinate{X: 1}}, Clear: true}},
	})
	view = economyView(4)
	view.resources[ai.ResourceFood] = 0
	view.buildings[1].ActiveOrders = []ai.ActiveBuildingOrder{{ID: 701, Kind: ai.BuildingOrderCreate, ItemID: 2}}
	view.buildings[1].Idle = false
	view.buildings[0].ResourcesLeft, view.buildings[0].Renewing = 0, true
	view.buildings[0].RenewCost = ai.Cost{Wood: 99}
	assertScenarioCalls(t, "cancellation and renewal", model.DecideOrders(view), view, []namedCall{
		{"refund once", orderCall{Kind: 20, Target: 701}},
		{"reuse refund", orderCall{Kind: 13, Subject: 801, Item: 2}},
		{"already paid renewal", orderCall{Kind: 5, Subject: 91, Target: 800, Place: ai.ZoneRef{Zone: ai.Coordinate{X: 1}}, Clear: true}},
		{"refund consumed and renewal free", orderCall{Kind: 1, Subject: 99, Place: ai.ZoneRef{Space: ai.Coordinate{Y: 30}}, Clear: true}},
	})
	checkEconomySelections(t, model)
}

func checkEconomySelections(t *testing.T, model ai.Model) {
	t.Helper()
	for _, penalty := range []float64{-1, 0, 1} {
		view := economyView(5)
		view.resources[ai.ResourceStone] = penalty
		view.knownResources = append(view.knownResources, ai.ResourceView{InternalID: 402, Category: ai.ResourceFood, Location: ai.ZoneRef{Space: ai.Coordinate{X: 2}}, AmountLeft: 10})
		view.units[1].CurrentOrder = &ai.CurrentOrder{Kind: ai.OrderKindGather, TargetResource: &view.knownResources[0]}
		expected := []namedCall{{"idle worker follows demand", orderCall{Kind: 2, Subject: 91, Target: 402, Place: ai.ZoneRef{Space: ai.Coordinate{X: 2}}, Clear: true}}}
		if penalty < 0 {
			expected = append(expected, namedCall{"reassign only above threshold", orderCall{Kind: 2, Subject: 92, Target: 402, Place: ai.ZoneRef{Space: ai.Coordinate{X: 2}}, Clear: true}})
		}
		assertScenarioCalls(t, "balanced gathering", model.DecideOrders(view), view, expected)
	}
	for _, minimum := range []float64{3, 3.1} {
		view := economyView(6)
		for _, query := range []string{"BuildSites", "CountSpaces", "EnemyDistance", "RegionProgress"} {
			view.allowed[query] = true
		}
		view.resources[ai.ResourceGold] = minimum
		view.sites = []ai.ZoneRef{{Space: ai.Coordinate{X: 2}}, {Space: ai.Coordinate{X: 3}}}
		view.enemyDistances = map[ai.Coordinate]int{{X: 2}: 2, {X: 3}: 3}
		view.regionProgress = map[ai.Coordinate]float64{{X: 2}: 0.5}
		view.countSpaces = func(where ai.SpaceCondition, include func(ai.Coordinate) bool) int {
			if where.Owner != "unowned" {
				t.Errorf("build-site predicate: %+v", where)
			}
			include(ai.Coordinate{X: 2})
			include(ai.Coordinate{X: 3})
			return 0
		}
		var expected []namedCall
		wood := 30
		if minimum == 3 {
			wood = 25
			for _, subject := range []uint64{91, 92} {
				expected = append(expected, namedCall{"best site shares construction cost", orderCall{Kind: 3, Subject: subject, Item: 2, Place: ai.ZoneRef{Space: ai.Coordinate{X: 3}}, Clear: true}})
			}
		}
		expected = append(expected, namedCall{"construction budget", orderCall{Kind: 1, Subject: 99, Place: ai.ZoneRef{Space: ai.Coordinate{X: wood, Y: 9}}, Clear: true}})
		assertScenarioCalls(t, "scored build sites", model.DecideOrders(view), view, expected)
	}
}

func TestCombatScenario(t *testing.T) {
	model, view := requireModel(t, combatScenario), combatView()
	var expected []namedCall
	for round, target := range []uint64{201, 202} {
		for _, subject := range []uint64{94, 93} {
			expected = append(expected, namedCall{"stamina priority and queued target", orderCall{Kind: 6, Subject: subject, Target: target, Place: ai.ZoneRef{Space: ai.Coordinate{X: 4 + round}}, Clear: round == 0}})
		}
	}
	expected = append(expected, namedCall{"building range", orderCall{Kind: 16, Subject: 800, Target: 201, Place: ai.ZoneRef{Space: ai.Coordinate{X: 4}}}}, namedCall{"restored scoring context", orderCall{Kind: 1, Subject: 99, Clear: true}})
	assertScenarioCalls(t, "scored queued attacks", model.DecideOrders(view), view, expected)
	view = combatView()
	view.resources[ai.ResourceGold] = 0.81
	assertScenarioCalls(t, "above minimum", model.DecideOrders(view), view, expected[4:])
	view = combatView()
	view.resources[ai.ResourceGold], view.resources[ai.ResourceStone] = 0.8, 1
	view.attackers = map[attackKey]map[uint64]bool{{ID: 201}: {94: true}}
	assertScenarioCalls(t, "boundary score and reassignment exclusion", model.DecideOrders(view), view, append([]namedCall{{"reassign self at capacity", orderCall{Kind: 6, Subject: 94, Target: 201, Place: ai.ZoneRef{Space: ai.Coordinate{X: 4}}, Clear: true}}}, expected[4:]...))
	view = combatView()
	view.units[0].CurrentOrder = &ai.CurrentOrder{Kind: ai.OrderKindAttackUnit}
	assertScenarioCalls(t, "preserve existing fight", model.DecideOrders(view), view, []namedCall{expected[0], expected[2], expected[4], expected[5]})
	view = combatView()
	view.resources[ai.ResourceStone] = 1
	view.attackers = map[attackKey]map[uint64]bool{{ID: 201}: {777: true}}
	firstAttack := expected[2]
	firstAttack.call.Clear = true
	assertScenarioCalls(t, "external attacker fills target", model.DecideOrders(view), view, []namedCall{firstAttack, expected[4], expected[5]})
	view = combatView()
	view.visibleEnemyUnits = nil
	view.visibleEnemyBuildings = []ai.BuildingView{}
	assertScenarioCalls(t, "remembered building outside visibility", model.DecideOrders(view), view, expected[5:])
	view = combatView()
	view.visibleEnemyUnits = nil
	view.attackRange = func(ai.BuildingView, ai.ZoneRef) bool { return true }
	assertScenarioCalls(t, "visible building fallback", model.DecideOrders(view), view, []namedCall{{"building target", orderCall{Kind: 17, Subject: 800, Target: 301, Place: ai.ZoneRef{Space: ai.Coordinate{X: 5}}}}, expected[5]})
}

func TestSpatialScenario(t *testing.T) {
	model, view := requireModel(t, spatialScenario), spatialView()
	view.closestSpaces = func(from ai.Coordinate, where ai.SpaceCondition, excluded bool) []ai.Coordinate {
		if from != (ai.Coordinate{}) || where.Vision == nil || *where.Vision != 0 || where.Owner != "unowned" || !excluded {
			t.Errorf("closest-space arguments: %+v/%+v/%v", from, where, excluded)
		}
		return []ai.Coordinate{{X: 2}}
	}
	expected := []namedCall{
		{"selected coordinates and score", orderCall{Kind: 1, Subject: 99, Place: ai.ZoneRef{Space: ai.Coordinate{X: 4, Y: 8}}, Clear: true}},
		{"distance two advances", orderCall{Kind: 1, Subject: 93, Place: ai.ZoneRef{Space: ai.Coordinate{X: 2}, Zone: ai.Coordinate{X: 1}}, Clear: true}},
		{"center advances", orderCall{Kind: 1, Subject: 94, Place: ai.ZoneRef{Space: ai.Coordinate{X: 2}, Zone: ai.Coordinate{X: 1}}, Clear: true}},
		{"distance three regroups", orderCall{Kind: 1, Subject: 95, Place: ai.ZoneRef{Space: ai.Coordinate{X: 2}}, Clear: true}},
	}
	assertScenarioCalls(t, "selection and facing", model.DecideOrders(view), view, expected)
	view = spatialView()
	view.visibleEnemyUnits, view.knownEnemyBuildings = nil, nil
	assertScenarioCalls(t, "no candidate preserves selection", model.DecideOrders(view), view, expected[:1])
	view = spatialView()
	view.visibleEnemyUnits, view.knownEnemyBuildings = nil, nil
	assertScenarioCalls(t, "independent model has no selection", requireModel(t, spatialScenario).DecideOrders(view), view, []namedCall{{"isolated variables", orderCall{Kind: 1, Subject: 99, Clear: true}}})
	view = spatialView()
	view.closestSpaces = func(ai.Coordinate, ai.SpaceCondition, bool) []ai.Coordinate { return []ai.Coordinate{{X: -5}} }
	view.spaceDistances[[2]ai.Coordinate{{X: -5}, {X: 3}}] = 1
	seamExpected := append([]namedCall(nil), expected...)
	for index := 1; index < 3; index++ {
		seamExpected[index].call.Place = ai.ZoneRef{Space: ai.Coordinate{X: -5}}
	}
	assertScenarioCalls(t, "graph seam cannot create invalid zone", model.DecideOrders(view), view, seamExpected)
	checkSpatialFilters(t)
	checkExploration(t)
}

func checkExploration(t *testing.T) {
	t.Helper()
	for _, stage := range []string{"seeded ties", "claimed destinations", "fallback", "empty"} {
		model := actionsModel(t, `{"action":"explore","unit_ids":[2],"anchor":"home","min_distance":2,"max":2}`)
		view := economyView(7)
		view.allowed["AllSpaces"], view.allowed["NearestUnexplored"] = true, true
		view.spaceDistances[[2]ai.Coordinate{{}, {X: -2}}] = 2
		view.allSpaces = []ai.SpaceInfo{{Space: ai.Coordinate{X: 2}}, {Space: ai.Coordinate{X: -2}}}
		targets := []int{-2, 2}
		switch stage {
		case "claimed destinations":
			place := ai.ZoneRef{Space: ai.Coordinate{X: 2}}
			view.units[4].CurrentOrder = &ai.CurrentOrder{Kind: ai.OrderKindMove, TargetZone: &place}
			targets = []int{-2, -2}
		case "fallback":
			view.allSpaces = nil
			view.unexplored = []ai.ZoneRef{{Space: ai.Coordinate{X: 3}}}
			targets = []int{3, 3}
		case "empty":
			view.allSpaces, targets = nil, nil
		}
		var expected []namedCall
		for index, target := range targets {
			expected = append(expected, namedCall{stage, orderCall{Kind: 1, Subject: uint64(93 + index), Place: ai.ZoneRef{Space: ai.Coordinate{X: target}}, Clear: true}})
		}
		assertScenarioCalls(t, stage, model.DecideOrders(view), view, expected)
	}
}

func checkSpatialFilters(t *testing.T) {
	t.Helper()
	for _, direction := range []string{"toward", "away"} {
		for _, held := range []string{"", `,"held_by_bucket":"guard"`} {
			model := actionsModel(t, `{"action":"set_bucket","bucket":"guard","size":1,"task":{"action":"delete_unit"}},{"action":"fill_bucket","bucket":"guard","unit_ids":[2]},{"action":"move","unit_ids":[99],"targets":"closest_spaces","from":{"x":0,"y":0},"direction_from":{"x":4,"y":0},"direction":"`+direction+`","exclude_reference":true`+held+`}`)
			view := spatialView()
			view.units[0].Location.Space.X = 2
			view.allowed["CountSpaces"] = true
			view.countSpaces = func(where ai.SpaceCondition, include func(ai.Coordinate) bool) int {
				for _, x := range []int{0, -5, 2, 3, 5} {
					include(ai.Coordinate{X: x})
				}
				return 0
			}
			var expected []namedCall
			if direction == "toward" || held == "" {
				x := 2
				if direction == "away" {
					x = -5
				}
				expected = append(expected, namedCall{"direction and held-space filter", orderCall{Kind: 1, Subject: 99, Place: ai.ZoneRef{Space: ai.Coordinate{X: x}}, Clear: true}})
			}
			assertScenarioCalls(t, direction+held, model.DecideOrders(view), view, expected)
		}
	}
}

func assertScenarioCalls(t *testing.T, stage string, decision ai.Decision, view *fakeView, expected []namedCall) {
	t.Helper()
	if len(decision.Behaviors)+len(decision.BuildingBehaviors) != 0 {
		t.Errorf("%s/unrequested behaviors: %+v/%+v", stage, decision.Behaviors, decision.BuildingBehaviors)
	}
	if len(view.calls) != len(expected) {
		t.Errorf("%s/constructors: got %d, want %d: %+v", stage, len(view.calls), len(expected), view.calls)
	}
	if len(decision.Orders) != len(expected) {
		t.Errorf("%s/orders: got %d, want %d", stage, len(decision.Orders), len(expected))
	}
	for index, want := range expected {
		if index < len(decision.Orders) {
			order := ai.Order{Subjects: []uint64{want.call.Subject}, OrderType: want.call.Kind, TargetID: int64(index + 1), ClearPreviousOrders: want.call.Clear}
			if !reflect.DeepEqual(decision.Orders[index], order) {
				t.Errorf("%s/%s/delivery: got %+v, want %+v", stage, want.label, decision.Orders[index], order)
			}
		}
		if index >= len(view.calls) {
			t.Errorf("%s/%s: missing constructor", stage, want.label)
			continue
		}
		got := view.calls[index]
		for _, field := range []struct {
			label     string
			got, want any
		}{{"kind", got.Kind, want.call.Kind}, {"subject", got.Subject, want.call.Subject}, {"target", got.Target, want.call.Target}, {"item", got.Item, want.call.Item}, {"space", got.Place.Space, want.call.Place.Space}, {"zone", got.Place.Zone, want.call.Place.Zone}, {"clear", got.Clear, want.call.Clear}} {
			if !reflect.DeepEqual(field.got, field.want) {
				t.Errorf("%s/%s/%s: got %+v, want %+v", stage, want.label, field.label, field.got, field.want)
			}
		}
	}
}
