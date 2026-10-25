package unit

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/ai"
)

func productionView() *fakeView {
	view := checkedView("Buildings", "AllBuildings", "AllUnits", "AllSpaces", "AllRegions", "KnownResources", "IdleUnits", "IdleBuildings", "ScopedUnits", "EligibleUnits", "VisibleEnemyUnits", "KnownEnemyBuildings")
	view.knownEnemyBuildings[0].PlayerID = 1
	second := view.buildings[0]
	second.InternalID, second.Location = 801, ai.ZoneRef{Zone: ai.Coordinate{X: 2}}
	view.buildings = append(view.buildings, second)
	return view
}

type queueTestView struct {
	*fakeView
	entries  []ai.OrderEntry
	revision uint64
}

func (v *queueTestView) OrderQueue() []ai.OrderEntry      { return v.entries }
func (v *queueTestView) Issue(r ai.IssueRequest) ai.Order { return ai.DispatchIssue(v, r) }
func (v *queueTestView) Revision() uint64                 { return v.revision }

func TestOrdersCountDistinctSubjectsAndKinds(t *testing.T) {
	v := &queueTestView{fakeView: productionView(), entries: []ai.OrderEntry{
		{SubjectUID: 91, Kind: ai.QueueOrderNames["attack_unit"], TargetUID: 60},
		{SubjectUID: 91, Kind: ai.QueueOrderNames["auto_attack_unit"], TargetUID: 60, QueuePosition: 1},
		{SubjectUID: 92, Kind: ai.QueueOrderNames["attack_space"], TargetUID: 60},
	}}
	config := rule(`{"action":"set_var","name":"n","value":{"count":{"kind":"orders","where":{"in":{"order_kind":["attack_unit","auto_attack_unit"]},"if":"var(item.target_uid) == 60"},"distinct":"subject_uid"}}}`)
	if got := decide(t, config, v, "n")["n"]; got != 1 {
		t.Fatalf("count = %v, want 1", got)
	}
}

func TestOrdersSubjectSets(t *testing.T) {
	v := &queueTestView{fakeView: productionView(), entries: []ai.OrderEntry{{SubjectUID: 91}, {SubjectUID: 1_000_000_800, SubjectKind: 1}}}
	for _, kind := range []string{"units", "buildings"} {
		config := rule(`{"action":"select","pick":{"kind":"` + kind + `"},"into":{"set":"subjects"}},
			{"action":"set_var","name":"included","value":{"count":{"kind":"orders","where":{"subject_in_set":"subjects"}}}},
			{"action":"set_var","name":"excluded","value":{"count":{"kind":"orders","where":{"subject_not_in_set":"subjects"}}}}`)
		got := decide(t, config, v, "included", "excluded")
		if got["included"] != 1 || got["excluded"] != 1 {
			t.Fatalf("%s: %v", kind, got)
		}
	}
}

func TestOrdersRejectSetsAndLocations(t *testing.T) {
	for _, where := range []string{`{"near":{"of":{"x":0,"y":0}}}`, `{"at":{}}`, `{"in_set":"x"}`, `{"not_in_set":"x"}`, `{"if":"var(target.x)"}`, `{"in":{"order_kind":["move"]}}`} {
		if loadError(rule(`{"action":"select","pick":{"kind":"orders","where":`+where+`}}`)) == nil {
			t.Errorf("accepted %s", where)
		}
	}
	if loadError(rule(`{"action":"select","pick":{"kind":"orders"},"into":{"set":"x"}}`)) == nil {
		t.Fatal("accepted order sink")
	}
}

func TestOrdersEqualityJoinStaysWithinBudget(t *testing.T) {
	v := &queueTestView{fakeView: productionView()}
	v.units = make([]ai.UnitView, 40)
	for i := range v.units {
		v.units[i].InternalID = uint64(i + 1)
	}
	for i := range 3000 {
		v.entries = append(v.entries, ai.OrderEntry{SubjectUID: uint64(i%40 + 1), QueuePosition: i / 40})
	}
	config := rule(`{"action":"set_var","name":"total","value":0},
		{"action":"for_each","source":"units","as":"u","do":[
		{"action":"set_var","name":"n","value":{"count":{"kind":"orders","where":{"if":"var(item.subject_uid) == var(u.uid)"}}}},
		{"action":"set_var","name":"total","value":"var(total) + var(n)"}]}`)
	if got := decide(t, config, v, "total")["total"]; got != 3000 {
		t.Fatalf("count = %v", got)
	}
}

func (v *queueTestView) MoveSpaceOrder(u ai.UnitView, target ai.Coordinate, clear bool) ai.Order {
	v.entries = append(v.entries, ai.OrderEntry{SubjectUID: u.InternalID, Kind: ai.QueueMoveSpace, Target: ai.ZoneRef{Space: target}})
	v.revision++
	return ai.Order{Subjects: []uint64{u.InternalID}}
}

func TestOrdersJoinRefreshesAfterEmission(t *testing.T) {
	v := &queueTestView{fakeView: productionView()}
	v.allowed = nil
	v.matchingSpaces = func(ai.SpaceCondition) []ai.Coordinate { return []ai.Coordinate{{X: 1}, {X: 2}} }
	config := rule(`{"action":"for_each","select":{"kind":"spaces","limit":2},"as":"s","do":[
		{"action":"select","actors":{"kind":"units"},"pick":{"kind":"spaces","limit":1},"issue":[{"move":{"to":"space"}}]},
		{"action":"set_var","name":"n","value":{"count":{"kind":"orders","where":{"if":"var(item.subject_uid) == 91"}}}}]}`)
	if got := decide(t, config, v, "n")["n"]; got != 2 {
		t.Fatalf("count = %v", got)
	}
}

func TestDistinctRejectsInvalidFieldsAndOtherAggregates(t *testing.T) {
	for _, query := range []string{`{"count":{"kind":"orders","distinct":"x"}}`, `{"count":{"kind":"orders","distinct":3}}`, `{"sum":{"kind":"units","of":1,"distinct":"uid"}}`} {
		if loadError(rule(`{"action":"set_var","name":"n","value":`+query+`}`)) == nil {
			t.Errorf("accepted %s", query)
		}
	}
}

func TestSelectStopRefundsSharedProductionAndRefreshesQueues(t *testing.T) {
	view := productionView()
	view.allowed = nil
	view.popCurrent, view.popLimit = 0, 10
	view.resources[ai.ResourceFood] = 20
	shared := ai.ActiveBuildingOrder{ID: 700, Kind: ai.BuildingOrderCreate, ItemID: 2}
	view.buildings[0].ActiveOrders = []ai.ActiveBuildingOrder{shared, {ID: 701, Kind: ai.BuildingOrderResearch, ItemID: 3}}
	view.buildings[1].ActiveOrders = []ai.ActiveBuildingOrder{shared}
	view.buildings[0].Idle, view.buildings[1].Idle = false, false
	model := actionsModel(t, `{"action":"select","actors":{"kind":"buildings","where":{"in":{"uid":[1000000800]}}},"issue":[{"stop":{"orders":["create"],"item_ids":[2]}}]},
		{"action":"select","actors":{"kind":"buildings","where":{"in":{"idle":[1]},"can_create":2}},"issue":[{"create":{"unit_id":2}}]},
		{"action":"move","targets":"coordinate","coordinate":{"x":"food + 100 * population","y":0}}`)
	assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: 20, Target: 700}, orderCall{Kind: 13, Subject: 801, Item: 2}, orderCall{Kind: 1, Subject: 91, Place: ai.ZoneRef{Space: ai.Coordinate{X: 123}}, Clear: true})
}

func TestSelectProductionSkipsQueueEvaluationForIneligibleBuildings(t *testing.T) {
	for _, id := range []uint32{2, 99} {
		view := productionView()
		if id == 2 {
			for i := range view.buildings {
				view.buildings[i].UnderConstruction = true
			}
		}
		var diagnostics bytes.Buffer
		config := rule(fmt.Sprintf(`{"action":"select","actors":{"kind":"buildings"},"issue":[{"create":{"unit_id":%d,"queue":"1 / 0"}}]}`, id))
		decision := jsonModel(t, config, &diagnostics).DecideOrders(view)
		if len(decision.Orders) != 0 || diagnostics.Len() != 0 {
			t.Errorf("unit %d: got orders %+v and diagnostics %q, want neither", id, decision.Orders, diagnostics.String())
		}
	}
}

func TestSelectCreateOrdersOneUnitPerMatchingBuilding(t *testing.T) {
	model := actionsModel(t, `{"action":"select","actors":{"kind":"buildings","where":{"can_create":2}},"issue":[{"create":{"unit_id":2}}]}`)
	view := productionView()
	assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: 13, Subject: 800, Item: 2}, orderCall{Kind: 13, Subject: 801, Item: 2})
}

func TestSelectResearchIsOrderedOnceAcrossBuildings(t *testing.T) {
	model := actionsModel(t, `{"action":"select","actors":{"kind":"buildings","where":{"can_research":3}},"issue":[{"research":{"tech_id":3}}]}`)
	view := productionView()
	assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: 14, Subject: 800, Item: 3})
}

func TestSelectProductionSkipsBuildingsThatCannotProduce(t *testing.T) {
	view := productionView()
	view.buildings[0].UnderConstruction = true
	model := actionsModel(t, `{"action":"select","actors":{"kind":"buildings"},"issue":[{"create":{"unit_id":2}},{"create":{"unit_id":99}}]}`)
	assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: 13, Subject: 801, Item: 2})
}

func TestSelectCreateQueueTopsUpToDepth(t *testing.T) {
	view := productionView()
	view.buildings[0].PlannedProduction = 2
	model := actionsModel(t, `{"action":"select","actors":{"kind":"buildings","where":{"building_ids":[1]},"rank":"0 - var(actor.uid)","limit":2},"issue":[{"create":{"unit_id":2,"queue":3}}]}`)
	assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: 13, Subject: 800, Item: 2}, orderCall{Kind: 13, Subject: 801, Item: 2}, orderCall{Kind: 13, Subject: 801, Item: 2}, orderCall{Kind: 13, Subject: 801, Item: 2})
}

func TestSelectBuildingIdleAndQueueLengthFields(t *testing.T) {
	view := productionView()
	view.buildings[0].PlannedProduction = 1
	model := actionsModel(t, `{"action":"select","actors":{"kind":"buildings","where":{"in":{"idle":[1]},"if":"1 - var(actor.queue_length)"}},"issue":[{"create":{"unit_id":2}}]}`)
	assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: 13, Subject: 801, Item: 2})
}

func TestSelectProductionChargesCostAndPopulation(t *testing.T) {
	view := productionView()
	view.resources[ai.ResourceFood] = 6
	view.popCurrent, view.popLimit = 0, 10
	create := `{"action":"select","actors":{"kind":"buildings","rank":"0 - var(actor.uid)","limit":1},"issue":[{"create":{"unit_id":2}}]}`
	model := actionsModel(t, `{"action":"select","actors":{"kind":"buildings"},"issue":[{"create":{"unit_id":2}}]},
		{"action":"if","when":{"resource_at_least":{"category":"food","amount":1}},"then":[`+create+`]},
		{"action":"if","when":{"population_headroom_equals":{"amount":8}},"then":[`+create+`]}`)
	calls := []orderCall{{Kind: 13, Subject: 800, Item: 2}, {Kind: 13, Subject: 801, Item: 2}, {Kind: 13, Subject: 800, Item: 2}}
	assertCalls(t, model.DecideOrders(view), view, calls...)
}

func TestSelectProductionConfigErrors(t *testing.T) {
	for name, action := range map[string]string{
		"mixed verbs":            `{"action":"select","actors":{"kind":"buildings"},"issue":[{"create":{"unit_id":2}},{"delete":{}}]}`,
		"no actors":              `{"action":"select","issue":[{"create":{"unit_id":2}}]}`,
		"unit actors":            `{"action":"select","actors":{"kind":"units"},"issue":[{"create":{"unit_id":2}}]}`,
		"with assign":            `{"action":"select","actors":{"kind":"buildings"},"assign":{"capacity":1},"issue":[{"research":{"tech_id":3}}]}`,
		"with mode":              `{"action":"select","actors":{"kind":"buildings"},"mode":"append","issue":[{"create":{"unit_id":2}}]}`,
		"with pick":              `{"action":"select","actors":{"kind":"buildings"},"pick":{"kind":"zones"},"issue":[{"create":{"unit_id":2}}]}`,
		"missing id":             `{"action":"select","actors":{"kind":"buildings"},"issue":[{"create":{}}]}`,
		"wrong id key":           `{"action":"select","actors":{"kind":"buildings"},"issue":[{"research":{"unit_id":3}}]}`,
		"can_create on units":    `{"action":"select","actors":{"kind":"units","where":{"can_create":2}},"issue":[{"move":{}}],"pick":{"kind":"zones"}}`,
		"bad can_research":       `{"action":"select","actors":{"kind":"buildings","where":{"can_research":0}},"issue":[{"create":{"unit_id":2}}]}`,
		"can_build on buildings": `{"action":"select","actors":{"kind":"buildings","where":{"can_build":2}},"issue":[{"create":{"unit_id":2}}]}`,
		"bad can_build":          `{"action":"select","actors":{"kind":"units","where":{"can_build":0}},"issue":[{"move":{}}],"pick":{"kind":"zones"}}`,
	} {
		if loadError(rule(action)) == nil {
			t.Errorf("%s: malformed select accepted", name)
		}
	}
}

func TestSelectCanBuildFiltersUnits(t *testing.T) {
	view := productionView()
	worker := ai.UnitView{InternalID: 10, Type: ai.UnitTypeEconomic, Builds: []ai.Producible{{Kind: ai.ProducibleBuilding, ID: 5}}}
	soldier := ai.UnitView{InternalID: 11, Type: ai.UnitTypeInfantry}
	view.units = []ai.UnitView{worker, soldier}
	model := actionsModel(t, `{"action":"select","actors":{"kind":"units","where":{"can_build":5}},"issue":[{"delete":{}}]}`)
	assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: 12, Subject: 10, Clear: true})
}

func TestSelectCreateQueueSupportsActorLets(t *testing.T) {
	view := productionView()
	model := actionsModel(t, `{"action":"select","actors":{"kind":"buildings","rank":"0 - var(actor.uid)","limit":1,"let":{"depth":2}},"issue":[{"create":{"unit_id":2,"queue":"var(actor.depth)"}}]}`)
	assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: 13, Subject: 800, Item: 2}, orderCall{Kind: 13, Subject: 800, Item: 2})
}

func TestSelectQueueExpressionsSeeEarlierProductionVerbs(t *testing.T) {
	for _, queue := range []string{"var(actor.queue_length) + 1", "var(actor.next_depth)"} {
		view := productionView()
		view.buildings = view.buildings[:1]
		model := actionsModel(t, fmt.Sprintf(`{"action":"select","actors":{"kind":"buildings","let":{"next_depth":"var(actor.queue_length) + 1"}},
			"issue":[{"create":{"unit_id":2}},{"create":{"unit_id":2,"queue":%q}}]}`, queue))
		assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: 13, Subject: 800, Item: 2}, orderCall{Kind: 13, Subject: 800, Item: 2})
	}
}

func TestSelectCanCreateExcludesUnderConstructionWithLimit(t *testing.T) {
	view := productionView()
	view.buildings[0].UnderConstruction = true
	model := actionsModel(t, `{"action":"select","actors":{"kind":"buildings","where":{"can_create":2},"limit":1},"issue":[{"create":{"unit_id":2}}]}`)
	assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: 13, Subject: 801, Item: 2})
}
