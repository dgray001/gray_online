package unit

import (
	"bytes"
	"encoding/json"
	"maps"
	"strings"
	"testing"

	. "github.com/dgray001/gray_online/game/games/risq/ai"
)

func parseDiagnostics(config string) string {
	var raw map[string]any
	json.Unmarshal([]byte(config), &raw)
	var buf bytes.Buffer
	ParseModelWithDiagnostics(raw, nil, &buf)
	return buf.String()
}

func TestSelectLegacyAssignRejected(t *testing.T) {
	config := rule(`{"action":"select","pick":{"kind":"spaces"},"assign":{"capacity":1},"issue":[{"move":{}}]}`)
	diag := parseDiagnostics(config)
	if !strings.Contains(diag, "assign was replaced by pick.capacity and match") {
		t.Fatalf("expected legacy assign rejection error, got: %s", diag)
	}
}

func TestSelectMatchRejectsAggregates(t *testing.T) {
	for _, bad := range []string{
		`{"action":"select","actors":{"kind":"units"},"pick":{"kind":"spaces"},"match":{"where":{"if":{"count":{"kind":"units"}}}},"issue":[{"move":{}}]}`,
		`{"action":"select","actors":{"kind":"units"},"pick":{"kind":"spaces"},"match":{"weight":{"count":{"kind":"units"}}},"issue":[{"move":{}}]}`,
		`{"action":"select","actors":{"kind":"units"},"pick":{"kind":"spaces"},"match":{"score":[{"count":{"kind":"units"}}]},"issue":[{"move":{}}]}`,
	} {
		diag := parseDiagnostics(rule(bad))
		if !strings.Contains(diag, "cannot contain aggregates") {
			t.Fatalf("expected aggregate rejection in match for %s, got: %s", bad, diag)
		}
	}
}

func TestSelectMatchFieldScopeEnforcement(t *testing.T) {
	for _, bad := range []string{
		`{"action":"select","pick":{"kind":"spaces","capacity":"var(target.assigned)"},"issue":[{"move":{}}]}`,
		`{"action":"select","pick":{"kind":"spaces","let":{"x":"var(target.assigned)"}},"issue":[{"move":{}}]}`,
		`{"action":"select","pick":{"kind":"spaces","let":{"x":"var(target.remaining)"}},"issue":[{"move":{}}]}`,
		`{"action":"select","pick":{"kind":"spaces","let":{"x":"var(target.already_targeted)"}},"issue":[{"move":{}}]}`,
		`{"action":"select","pick":{"kind":"spaces","where":{"if":"var(target.distance_to_actor)"}},"issue":[{"move":{}}]}`,
		`{"action":"select","actors":{"kind":"units","where":{"if":"var(actor.distance_to_target)"}},"pick":{"kind":"spaces"},"issue":[{"move":{}}]}`,
	} {
		if err := loadError(rule(bad)); err == nil {
			t.Fatalf("expected scope rejection error for: %s", bad)
		}
	}
}

func TestMatchCapacityOvershootAndPruning(t *testing.T) {
	u1, u2 := villager(1, Coordinate{}), villager(2, Coordinate{})
	u1.CanAttack, u2.CanAttack = true, true
	b1 := BuildingView{InternalID: 10, PlayerID: 1, Health: 3, MaxHealth: 100, Location: ZoneRef{Space: Coordinate{X: 1}}}
	bZero := BuildingView{InternalID: 20, PlayerID: 1, Health: 0, MaxHealth: 100, Location: ZoneRef{Space: Coordinate{X: 2}}}
	view := selectTestView{
		loopTestView: loopTestView{units: []UnitView{u1, u2}, enemy_buildings: []BuildingView{b1, bZero}},
		idle:         []UnitView{u1, u2}, turn: 1,
	}
	config := rule(`{"action":"select","actors":{"kind":"units","rank":"0 - var(actor.uid)"},
		"pick":{"kind":"buildings","where":{"owner":"enemy"},"capacity":"var(target.health)"},
		"match":{"weight":5},
		"issue":[{"attack":{}}]}`)
	orders := selectOrders(t, config, view)
	if len(orders) != 1 || orders[0].Subjects[0] != 1 || orders[0].TargetID != 10 {
		t.Fatalf("expected only unit 1 to claim building 10 (overshooting 3 HP with weight 5), got: %+v", orders)
	}
}

func TestMatchStrictFitViaRemaining(t *testing.T) {
	u1 := villager(1, Coordinate{})
	u1.CanAttack = true
	b1 := BuildingView{InternalID: 10, PlayerID: 1, Health: 3, MaxHealth: 100, Location: ZoneRef{Space: Coordinate{X: 1}}}
	view := selectTestView{
		loopTestView: loopTestView{units: []UnitView{u1}, enemy_buildings: []BuildingView{b1}},
		idle:         []UnitView{u1}, turn: 1,
	}
	config := rule(`{"action":"select","actors":{"kind":"units"},
		"pick":{"kind":"buildings","where":{"owner":"enemy"},"capacity":"var(target.health)"},
		"match":{"where":{"if":"var(target.remaining) >= 5"},"weight":5},
		"issue":[{"attack":{}}]}`)
	orders := selectOrders(t, config, view)
	if len(orders) != 0 {
		t.Fatalf("expected strict-fit rule to reject target with remaining 3 < weight 5, got: %+v", orders)
	}
}

func TestMatchLexicographicScore(t *testing.T) {
	units := []UnitView{villager(1, Coordinate{}), villager(2, Coordinate{}), villager(3, Coordinate{}), villager(4, Coordinate{})}
	s0 := SpaceInfo{Space: Coordinate{X: 0}}
	s1 := SpaceInfo{Space: Coordinate{X: 1}}
	view := selectTestView{
		loopTestView: loopTestView{units: units, spaces: []SpaceInfo{s0, s1}},
		idle:         units, turn: 1,
	}
	config := rule(`{"action":"select","actors":{"kind":"units","rank":"0 - var(actor.uid)"},
		"pick":{"kind":"spaces"},
		"match":{"score":["0 - var(target.assigned)", "0 - distance(actor, target)"]},
		"issue":[{"move":{}}]}`)
	orders := selectOrders(t, config, view)
	if len(orders) != 4 {
		t.Fatalf("expected 4 orders, got: %+v", orders)
	}
	counts := map[int64]int{}
	for _, o := range orders {
		counts[o.TargetID]++
	}
	if counts[0] != 2 || counts[100] != 2 {
		t.Fatalf("expected 2 orders on space 0 and 2 on space 1, got counts: %v", counts)
	}
}

func TestSelectCapacityVerbValidation(t *testing.T) {
	for _, bad := range []string{
		`{"action":"select","pick":{"kind":"spaces","capacity":1},"into":{"set":"s"}}`,
		`{"action":"select","pick":{"kind":"zones","capacity":1},"issue":[{"hire":{"unit_id":101,"count":1}}]}`,
		`{"action":"select","pick":{"kind":"foundations","capacity":1},"issue":[{"cancel_foundation":{}}]}`,
	} {
		diag := parseDiagnostics(rule(bad))
		if !strings.Contains(diag, "capacity is only valid on pick selectors with target-dependent verbs") {
			t.Fatalf("expected capacity verb rejection for %s, got: %s", bad, diag)
		}
	}
}

func TestMatchSecondaryRankKeysPreserved(t *testing.T) {
	u1 := villager(1, Coordinate{})
	s0 := SpaceInfo{Space: Coordinate{X: 0}}
	s1 := SpaceInfo{Space: Coordinate{X: 1}}
	view := selectTestView{
		loopTestView: loopTestView{units: []UnitView{u1}, spaces: []SpaceInfo{s0, s1}},
		idle:         []UnitView{u1}, turn: 1,
	}
	config := rule(`{"action":"select","actors":{"kind":"units"},
		"pick":{"kind":"spaces","rank":["0", "var(target.x)"]},
		"issue":[{"move":{}}]}`)
	orders := selectOrders(t, config, view)
	if len(orders) != 1 || orders[0].TargetID != 100 {
		t.Fatalf("expected secondary rank key to pick space 1 (targetID 100), got: %+v", orders)
	}
}

func TestMatchUncappedRemainingIsFinite(t *testing.T) {
	u1 := villager(1, Coordinate{})
	s0 := SpaceInfo{Space: Coordinate{X: 0}}
	view := selectTestView{
		loopTestView: loopTestView{units: []UnitView{u1}, spaces: []SpaceInfo{s0}},
		idle:         []UnitView{u1}, turn: 1,
	}
	config := rule(`{"action":"select","actors":{"kind":"units"},
		"pick":{"kind":"spaces"},
		"match":{"where":{"if":"var(target.remaining) > 1000"}},
		"issue":[{"move":{}}]}`)
	orders := selectOrders(t, config, view)
	if len(orders) != 1 {
		t.Fatalf("expected uncapped target.remaining to be finite and > 1000, got: %+v", orders)
	}
}

func enemyBuilding(id uint64, health float64, x int) BuildingView {
	return BuildingView{InternalID: id, PlayerID: 1, Health: health, MaxHealth: 100, Location: ZoneRef{Space: Coordinate{X: x}}}
}

func assignView(unit_count int, buildings ...BuildingView) selectTestView {
	units := make([]UnitView, unit_count)
	for i := range units {
		units[i] = villager(uint64(i+1), Coordinate{})
		units[i].CanAttack = true
	}
	return selectTestView{loopTestView: loopTestView{units: units, enemy_buildings: buildings}, idle: units, turn: 1}
}

const assignChainHead = `{"action":"select","actors":{"kind":"units","rank":"0 - var(actor.uid)"},"pick":{"kind":"buildings","where":{"owner":"enemy"},"capacity":"var(target.health)"},"match":{"score":"0 - var(target.uid)"},`

func TestAssignCarryRunsRoundsWithPairExclusionAndAppend(t *testing.T) {
	view := assignView(2, enemyBuilding(10, 5, 1), enemyBuilding(11, 5, 2))
	config := rule(assignChainHead + `"issue":[{"attack":{}}],"into":{"assign":"a"}},{"action":"select","match":{"carry":"a"}}`)
	orders := selectOrders(t, config, view)
	want := []struct {
		unit, target int64
		clear        bool
	}{{1, 10, true}, {2, 10, true}, {1, 11, false}, {2, 11, false}}
	if len(orders) != len(want) {
		t.Fatalf("expected %d orders, got: %+v", len(want), orders)
	}
	for i, w := range want {
		if int64(orders[i].Subjects[0]) != w.unit || orders[i].TargetID != w.target || orders[i].ClearPreviousOrders != w.clear {
			t.Fatalf("order %d = %+v, want unit %d target %d clear %v", i, orders[i], w.unit, w.target, w.clear)
		}
	}
}

func TestAssignRecordedLoadClosesTargetAcrossRounds(t *testing.T) {
	view := assignView(3, enemyBuilding(10, 2, 1))
	config := rule(assignChainHead + `"issue":[{"attack":{}}],"into":{"assign":"a"}},{"action":"select","match":{"carry":"a"}}`)
	orders := selectOrders(t, config, view)
	if len(orders) != 2 || orders[0].Subjects[0] != 1 || orders[1].Subjects[0] != 2 {
		t.Fatalf("expected units 1 and 2 to fill the 2 HP target and unit 3 to get nothing in either round, got: %+v", orders)
	}
}

func TestAssignKeepsCandidateIdentityWhenOrdersShareASpace(t *testing.T) {
	view := assignView(1, enemyBuilding(10, 5, 1), enemyBuilding(11, 5, 1))
	config := rule(assignChainHead + `"issue":[{"attack":{"to":"space"}}],"into":{"assign":"a"}},{"action":"select","match":{"carry":"a"}}`)
	orders := selectOrders(t, config, view)
	if len(orders) != 2 || orders[0].OrderType != kindAttackSpace || orders[1].OrderType != kindAttackSpace {
		t.Fatalf("expected two attack-space orders, got: %+v", orders)
	}
	if orders[0].TargetID != orders[1].TargetID || !orders[0].ClearPreviousOrders || orders[1].ClearPreviousOrders {
		t.Fatalf("expected the same space twice, clearing only the first, got: %+v", orders)
	}
}

func TestAssignInitialLoadSeedsAssignedAndRemaining(t *testing.T) {
	view := assignView(2, enemyBuilding(10, 3, 1))
	pick := `"pick":{"kind":"buildings","where":{"owner":"enemy"},"initial_assigned":2,"capacity":"var(target.assigned) + 1"}`
	config := rule(`{"action":"select","actors":{"kind":"units","rank":"0 - var(actor.uid)"},` + pick + `,"match":{"weight":1},"issue":[{"attack":{}}],"into":{"assign":"a"}}`)
	orders := selectOrders(t, config, view)
	if len(orders) != 1 || orders[0].Subjects[0] != 1 {
		t.Fatalf("expected capacity 3 minus initial load 2 to admit exactly one unit, got: %+v", orders)
	}
}

func TestAssignInitialLoadReadsAnotherRelation(t *testing.T) {
	view := assignView(2, enemyBuilding(10, 2, 1))
	shared := `{"sum":{"kind":"assignments","where":{"relation":"a","if":"var(item.target_uid) == var(target.uid)"},"of":"var(item.weight)"}}`
	first := `{"action":"select","actors":{"kind":"units","where":{"if":"var(actor.uid) == 1"}},"pick":{"kind":"buildings","where":{"owner":"enemy"},"capacity":"var(target.health)"},"match":{"weight":2},"issue":[{"attack":{}}],"into":{"assign":"a"}}`
	second := `{"action":"select","actors":{"kind":"units","where":{"if":"var(actor.uid) == 2"}},"pick":{"kind":"buildings","where":{"owner":"enemy"},"initial_assigned":` + shared + `,"capacity":"var(target.health)"},"match":{"weight":1},"issue":[{"attack":{}}],"into":{"assign":"b"}}`
	orders := selectOrders(t, rule(first+","+second), view)
	if len(orders) != 1 || orders[0].Subjects[0] != 1 {
		t.Fatalf("expected the load recorded by relation a to close the target for relation b, got: %+v", orders)
	}
}

func TestAssignRoundBindingAndRelationQueries(t *testing.T) {
	view := assignView(1, enemyBuilding(10, 100, 1), enemyBuilding(11, 100, 2))
	query := func(of string) string {
		return `{"sum":{"kind":"assignments","where":{"relation":"a"},"of":"` + of + `"}}`
	}
	config := rule(`{"action":"select","actors":{"kind":"units"},"pick":{"kind":"buildings","where":{"owner":"enemy"},"capacity":"var(target.health)"},` +
		`"match":{"score":"0 - var(target.uid)","weight":"1 + 10 * var(actor.assign_round)"},"issue":[{"attack":{}}],"into":{"assign":"a"}},` +
		`{"action":"select","match":{"carry":"a"}},` +
		`{"action":"set_var","name":"total","value":` + query("var(item.weight)") + `},{"action":"set_var","name":"rounds","value":` + query("var(item.round)") + `},` +
		`{"action":"set_var","name":"actor_sum","value":` + query("var(item.actor_uid)") + `}`)
	got := decide(t, config, view, "total", "rounds", "actor_sum")
	if got["total"] != 12 || got["rounds"] != 1 || got["actor_sum"] != 2 {
		t.Fatalf("expected weights 1 + 11, rounds 0 + 1 and the actor uid twice, got: %v", got)
	}
}

func TestAssignCarryWithoutRelationIsAnErrorAndDoesNothing(t *testing.T) {
	var diagnostics bytes.Buffer
	model := jsonModel(t, rule(`{"action":"select","match":{"carry":"missing"}}`), &diagnostics)
	if orders := model.DecideOrders(assignView(1, enemyBuilding(10, 5, 1))).Orders; len(orders) != 0 {
		t.Fatalf("expected no orders, got: %+v", orders)
	}
	if !strings.Contains(diagnostics.String(), `carry names no assignment "missing"`) {
		t.Fatalf("expected a carry diagnostic, got: %q", diagnostics.String())
	}
}

func TestAssignRelationsAreClearedBetweenDecisions(t *testing.T) {
	var diagnostics bytes.Buffer
	init := `{"when":{"value_at_most":{"value":"gameVar(turn)","amount":1}},"then":[` + assignChainHead + `"issue":[{"attack":{}}],"into":{"assign":"a"}}]}`
	carry := `{"when":{"value_at_least":{"value":"gameVar(turn)","amount":2}},"then":[{"action":"select","match":{"carry":"a"}}]}`
	model := jsonModel(t, `{"rules":[`+init+`,`+carry+`]}`, &diagnostics)
	view := assignView(1, enemyBuilding(10, 5, 1))
	if orders := model.DecideOrders(view).Orders; len(orders) != 1 {
		t.Fatalf("expected the first decision to assign once, got: %+v", orders)
	}
	view.turn = 2
	if orders := model.DecideOrders(view).Orders; len(orders) != 0 || !strings.Contains(diagnostics.String(), "carry names no assignment") {
		t.Fatalf("expected the relation to be gone in the next decision, got orders %+v and %q", orders, diagnostics.String())
	}
}

func TestAssignEmptyInitializationReplacesEarlierRelation(t *testing.T) {
	view := assignView(1, enemyBuilding(10, 5, 1))
	empty := `{"action":"select","actors":{"kind":"units","where":{"if":"0"}},"pick":{"kind":"buildings","where":{"owner":"enemy"}},"match":{"weight":1},"issue":[{"attack":{}}],"into":{"assign":"a"}}`
	count := `{"action":"set_var","name":"n","value":{"count":{"kind":"assignments","where":{"relation":"a"}}}}`
	config := rule(assignChainHead + `"issue":[{"attack":{}}],"into":{"assign":"a"}},` + empty + `,` + count)
	if got := decide(t, config, view, "n"); got["n"] != 0 {
		t.Fatalf("expected the empty initialization to replace the earlier relation, got %d records", int(got["n"]))
	}
}

func TestAssignUnreadableFieldIsAnEvaluationError(t *testing.T) {
	view := assignView(1)
	view.spaces = []SpaceInfo{{Space: Coordinate{X: 1}}}
	init := `{"action":"set_var","name":"n","value":42},{"action":"select","actors":{"kind":"units"},"pick":{"kind":"spaces"},"match":{"weight":1},"issue":[{"move":{}}],"into":{"assign":"a"}},`
	for name, expr := range map[string]string{
		"var":      `{"sum":{"kind":"assignments","where":{"relation":"a"},"of":"var(item.target_uid)"}}`,
		"distinct": `{"count":{"kind":"assignments","where":{"relation":"a"},"distinct":"target_uid"}}`,
		"in":       `{"count":{"kind":"assignments","where":{"relation":"a","in":{"target_uid":[1]}}}}`,
	} {
		var diagnostics bytes.Buffer
		model := jsonModel(t, rule(init+`{"action":"set_var","name":"n","value":`+expr+`}`), &diagnostics)
		model.DecideOrders(view)
		if !strings.Contains(diagnostics.String(), "not readable for this assignment") {
			t.Fatalf("%s: expected unreadable field diagnostic, got: %q", name, diagnostics.String())
		}
		if got := decide(t, rule(init+`{"action":"set_var","name":"n","value":`+expr+`}`), view, "n"); got["n"] != 42 {
			t.Fatalf("%s: expected set_var to abort and keep n=42, got %v", name, got["n"])
		}
	}
}

func TestAssignParseRejections(t *testing.T) {
	actors, pick := `"actors":{"kind":"units"}`, `"pick":{"kind":"buildings","where":{"owner":"enemy"}}`
	for name, bad := range map[string]string{
		"carry with extra keys":      `{"action":"select",` + actors + `,"match":{"carry":"a"}}`,
		"assign plus carry":          `{"action":"select","match":{"carry":"a"},"into":{"assign":"a"}}`,
		"assign with another sink":   `{"action":"select",` + actors + `,` + pick + `,"match":{},"issue":[{"attack":{}}],"into":{"assign":"a","set":"s"}}`,
		"assign without match":       `{"action":"select",` + actors + `,` + pick + `,"issue":[{"attack":{}}],"into":{"assign":"a"}}`,
		"assign with mode":           `{"action":"select",` + actors + `,` + pick + `,"match":{},"mode":"append","issue":[{"attack":{}}],"into":{"assign":"a"}}`,
		"bad relation name":          `{"action":"select",` + actors + `,` + pick + `,"match":{},"issue":[{"attack":{}}],"into":{"assign":"1a"}}`,
		"initial outside assign":     `{"action":"select",` + actors + `,"pick":{"kind":"buildings","initial_assigned":1},"match":{},"issue":[{"attack":{}}]}`,
		"round in actors":            `{"action":"select","actors":{"kind":"units","rank":"var(actor.assign_round)"},` + pick + `,"match":{},"issue":[{"attack":{}}],"into":{"assign":"a"}}`,
		"assigned in initial":        `{"action":"select",` + actors + `,"pick":{"kind":"buildings","initial_assigned":"var(target.assigned)"},"match":{},"issue":[{"attack":{}}],"into":{"assign":"a"}}`,
		"assignments as a target":    `{"action":"select",` + actors + `,"pick":{"kind":"assignments","where":{"relation":"a"}},"issue":[{"attack":{}}]}`,
		"assignments near":           `{"action":"set_var","name":"n","value":{"count":{"kind":"assignments","where":{"relation":"a","near":{"distance":1,"of":{"x":0,"y":0}}}}}}`,
		"assignments without name":   `{"action":"set_var","name":"n","value":{"count":{"kind":"assignments"}}}`,
		"assign with non-order verb": `{"action":"select",` + actors + `,` + pick + `,"match":{},"issue":[{"attack":{}},{"delete":{}}],"into":{"assign":"a"}}`,
	} {
		if err := loadError(rule(bad)); err == nil {
			t.Errorf("%s: expected a load error for %s", name, bad)
		} else if want := assignRejectionFragments[name]; !strings.Contains(parseDiagnostics(rule(bad)), want) {
			t.Errorf("%s: diagnostics %q lack %q", name, parseDiagnostics(rule(bad)), want)
		}
	}
}

func TestAssignFirstSuccessInALaterRoundClears(t *testing.T) {
	view := assignView(1, enemyBuilding(10, 5, 1))
	late := `"match":{"where":{"if":"var(actor.assign_round) >= 1"}},`
	config := rule(`{"action":"select","actors":{"kind":"units"},"pick":{"kind":"buildings","where":{"owner":"enemy"}},` + late + `"issue":[{"attack":{}}],"into":{"assign":"a"}},{"action":"select","match":{"carry":"a"}}`)
	orders := selectOrders(t, config, view)
	if len(orders) != 1 || !orders[0].ClearPreviousOrders {
		t.Fatalf("expected the first success, in round 1, to clear the actor's old orders, got: %+v", orders)
	}
}

func TestAssignCapacityAdmissionRules(t *testing.T) {
	for name, c := range map[string]struct {
		pick   string
		weight string
		want   int
	}{
		"overshoot admits one":      {`"capacity":3`, "5", 1},
		"zero weight keeps it open": {`"capacity":1`, "0", 3},
		"no capacity stays open":    {``, "5", 3},
	} {
		view := assignView(3, enemyBuilding(10, 5, 1))
		extra := ""
		if c.pick != "" {
			extra = "," + c.pick
		}
		config := rule(`{"action":"select","actors":{"kind":"units"},"pick":{"kind":"buildings","where":{"owner":"enemy"}` + extra + `},"match":{"weight":` + c.weight + `},"issue":[{"attack":{}}],"into":{"assign":"a"}}`)
		if orders := selectOrders(t, config, view); len(orders) != c.want {
			t.Errorf("%s: want %d orders, got %+v", name, c.want, orders)
		}
	}
}

func TestAssignRelationsAndSetsHaveSeparateCaseSensitiveNames(t *testing.T) {
	view := assignView(1, enemyBuilding(10, 5, 1))
	set := `{"action":"select","pick":{"kind":"units","where":{"owner":"own"}},"into":{"set":"a"}}`
	initUpper := `{"action":"select","actors":{"kind":"units"},"pick":{"kind":"buildings","where":{"owner":"enemy"}},"match":{"weight":7},"issue":[{"attack":{}}],"into":{"assign":"A"}}`
	counts := `{"action":"set_var","name":"low","value":{"sum":{"kind":"assignments","where":{"relation":"a"},"of":"var(item.weight)"}}},` +
		`{"action":"set_var","name":"up","value":{"sum":{"kind":"assignments","where":{"relation":"A"},"of":"var(item.weight)"}}},` +
		`{"action":"set_var","name":"set_n","value":{"count":{"kind":"units","where":{"in_set":"a"}}}}`
	config := rule(assignChainHead + `"issue":[{"attack":{}}],"into":{"assign":"a"}},` + set + `,` + initUpper + `,` + counts)
	if got := decide(t, config, view, "low", "up", "set_n"); got["low"] != 1 || got["up"] != 7 || got["set_n"] != 1 {
		t.Fatalf("expected independent relations a=1, A=7 and set a=1, got %v", got)
	}
}

func TestAssignFailedEmissionRecordsNothing(t *testing.T) {
	u := villager(1, Coordinate{})
	s0 := SpaceInfo{Space: Coordinate{X: 1}}
	view := selectTestView{loopTestView: loopTestView{units: []UnitView{u}, spaces: []SpaceInfo{s0}}, idle: []UnitView{u}, turn: 1}
	config := rule(`{"action":"select","actors":{"kind":"units"},"pick":{"kind":"spaces","capacity":1},"match":{"weight":1},"issue":[{"gather":{}}],"into":{"assign":"a"}},` +
		`{"action":"set_var","name":"n","value":{"count":{"kind":"assignments","where":{"relation":"a"}}}}`)
	if orders := selectOrders(t, config, view); len(orders) != 0 {
		t.Fatalf("expected 0 orders when gather has no resource, got %+v", orders)
	}
	if got := decide(t, config, view, "n"); got["n"] != 0 {
		t.Fatalf("expected 0 recorded assignments on failed verb emission, got %v", got["n"])
	}
}

func TestAssignCoordinateFieldsAndKindFilters(t *testing.T) {
	u := villager(1, Coordinate{X: 2, Y: 3})
	u.Location.Zone = Coordinate{X: 4, Y: 5}
	u.CanAttack = true
	b := BuildingView{InternalID: 10, PlayerID: 1, Health: 5, MaxHealth: 10, Location: ZoneRef{Space: Coordinate{X: 6, Y: 7}, Zone: Coordinate{X: 8, Y: 9}}}
	s := SpaceInfo{Space: Coordinate{X: 6, Y: 7}}
	view := selectTestView{loopTestView: loopTestView{units: []UnitView{u}, enemy_buildings: []BuildingView{b}, spaces: []SpaceInfo{s}}, idle: []UnitView{u}, turn: 1}
	expr := `"var(item.actor_x)*1 + var(item.actor_y)*10 + var(item.actor_zone_x)*100 + var(item.actor_zone_y)*1000 + var(item.target_x)*10000 + var(item.target_y)*100000 + var(item.target_zone_x)*1000000 + var(item.target_zone_y)*10000000"`
	q := `{"sum":{"kind":"assignments","where":{"relation":"a","actor_kind":"units","target_kind":"buildings"},"of":` + expr + `}}`
	qSpace := `{"count":{"kind":"assignments","where":{"relation":"s","actor_kind":"units","target_kind":"spaces"}}}`
	qFiltered := `{"count":{"kind":"assignments","where":{"relation":"a","actor_kind":"buildings","target_kind":"spaces"}}}`
	moveSpace := `{"action":"select","actors":{"kind":"units"},"pick":{"kind":"spaces"},"match":{"weight":1},"issue":[{"move":{}}],"into":{"assign":"s"}}`
	config := rule(assignChainHead + `"issue":[{"attack":{}}],"into":{"assign":"a"}},` + moveSpace + `,{"action":"set_var","name":"coords","value":` + q + `},{"action":"set_var","name":"spaces","value":` + qSpace + `},{"action":"set_var","name":"filtered","value":` + qFiltered + `}`)
	if got := decide(t, config, view, "coords", "spaces", "filtered"); got["coords"] != 98765432 || got["spaces"] != 1 || got["filtered"] != 0 {
		t.Fatalf("expected coords=98765432, spaces=1 and filtered=0, got %v", got)
	}
}

func kindCount(relation string, target_kind string, name string) string {
	return `{"action":"set_var","name":"` + name + `","value":{"count":{"kind":"assignments","where":{"relation":"` + relation + `","target_kind":"` + target_kind + `"}}}}`
}

func TestAssignLoopReplacementAndSpaceZoneIdentity(t *testing.T) {
	u1, u2 := villager(1, Coordinate{}), villager(2, Coordinate{})
	s := SpaceInfo{Space: Coordinate{X: 1, Y: 2}}
	z := ZoneInfo{Location: ZoneRef{Space: Coordinate{X: 1, Y: 2}, Zone: Coordinate{}}}
	view := selectTestView{loopTestView: loopTestView{units: []UnitView{u1, u2}, spaces: []SpaceInfo{s}, zones: []ZoneInfo{z}}, idle: []UnitView{u1, u2}, turn: 1}
	repCfg := rule(`{"action":"for_each","select":{"kind":"units"},"as":"u","do":[{"action":"select","actors":{"kind":"units","where":{"if":"var(actor.uid) == var(u.uid)"}},"pick":{"kind":"spaces"},"match":{"weight":1},"issue":[{"move":{}}],"into":{"assign":"a"}}]},` +
		`{"action":"set_var","name":"n","value":{"count":{"kind":"assignments","where":{"relation":"a"}}}}`)
	if got := decide(t, repCfg, view, "n"); got["n"] != 1 {
		t.Fatalf("expected each iteration of for_each to replace relation a (leaving 1 record), got %v", got["n"])
	}
	shared := `{"sum":{"kind":"assignments","where":{"relation":"s","target_kind":"spaces","if":"var(item.target_x) == var(target.x)"},"of":"var(item.weight)"}}`
	szCfg := rule(`{"action":"select","actors":{"kind":"units","where":{"if":"var(actor.uid) == 1"}},"pick":{"kind":"spaces","capacity":1},"match":{"weight":1},"issue":[{"move":{}}],"into":{"assign":"s"}},` +
		`{"action":"select","actors":{"kind":"units","where":{"if":"var(actor.uid) == 2"}},"pick":{"kind":"zones","initial_assigned":` + shared + `,"capacity":1},"match":{"weight":1},"issue":[{"move":{}}],"into":{"assign":"z"}},` +
		`{"action":"set_var","name":"z_cnt","value":{"count":{"kind":"assignments","where":{"relation":"z","target_kind":"zones"}}}},` +
		`{"action":"select","actors":{"kind":"units","where":{"if":"var(actor.uid) == 2"}},"pick":{"kind":"zones"},"match":{"weight":1},"issue":[{"move":{}}],"into":{"assign":"z2"}},` +
		kindCount("s", "spaces", "s_space") + `,` + kindCount("s", "zones", "s_zone") + `,` + kindCount("z2", "zones", "z2_zone") + `,` + kindCount("z2", "spaces", "z2_space"))
	want := map[string]float64{"z_cnt": 0, "s_space": 1, "s_zone": 0, "z2_zone": 1, "z2_space": 0}
	if got := decide(t, szCfg, view, "z_cnt", "s_space", "s_zone", "z2_zone", "z2_space"); !maps.Equal(got, want) {
		t.Fatalf("expected the space load to close the zone and each relation to match only its own target kind, want %v got %v", want, got)
	}
}

func TestAssignAbortedInitDeletesRelation(t *testing.T) {
	var diagnostics bytes.Buffer
	view := assignView(1, enemyBuilding(10, 5, 1))
	abort := `{"action":"select","actors":{"kind":"units"},"pick":{"kind":"buildings","where":{"owner":"enemy"},"initial_assigned":-1},"match":{"weight":1},"issue":[{"attack":{}}],"into":{"assign":"a"}}`
	config := rule(assignChainHead + `"issue":[{"attack":{}}],"into":{"assign":"a"}},` + abort + `,{"action":"select","match":{"carry":"a"}}`)
	jsonModel(t, config, &diagnostics).DecideOrders(view)
	if !strings.Contains(diagnostics.String(), `carry names no assignment "a"`) {
		t.Fatalf("expected aborted re-init to delete relation a so carry errors, got: %q", diagnostics.String())
	}
}

var assignRejectionFragments = map[string]string{
	"carry with extra keys":      `carry takes no "actors"`,
	"assign plus carry":          `carry takes no "into"`,
	"assign with another sink":   "takes only a valid assignment name",
	"assign without match":       `needs "match"`,
	"assign with mode":           "no mode other than replace",
	"bad relation name":          "takes only a valid assignment name",
	"initial outside assign":     "initial_assigned only applies",
	"round in actors":            `no field "assign_round"`,
	"assigned in initial":        `no field "assigned"`,
	"assignments as a target":    "cannot be the target of a verb",
	"assignments near":           `do not support "near"`,
	"assignments without name":   `need a valid "relation" name`,
	"assign with non-order verb": "target-dependent order verbs only",
}
