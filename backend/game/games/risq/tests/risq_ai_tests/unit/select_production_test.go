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
