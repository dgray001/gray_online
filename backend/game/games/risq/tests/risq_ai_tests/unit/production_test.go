package unit

import (
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"testing"
)

func TestProductionAndHireShareLastPopulationSlot(t *testing.T) {
	for _, c := range []struct {
		actions string
		call    orderCall
	}{
		{`{"action":"create","unit_id":2},{"action":"hire","unit_id":2}`, orderCall{Kind: 13, Subject: 800, Item: 2}},
		{`{"action":"hire","unit_id":2},{"action":"create","unit_id":2}`, orderCall{Kind: 19, Item: 2, Place: ai.ZoneRef{Zone: ai.Coordinate{X: 1}}}},
	} {
		model := actionsModel(t, c.actions)
		for turn := 0; turn < 2; turn++ {
			view := checkedView("Buildings", "IdleBuildings", "UnitCost", "TechResearched")
			view.popLimit = 2
			assertCalls(t, model.DecideOrders(view), view, c.call)
		}
	}
}

func TestFoundationRefundAndReplacement(t *testing.T) {
	model := actionsModel(t, `{"action":"cancel_foundations","building_ids":[2]},{"action":"cancel_foundations","building_ids":[2]},{"action":"build","building_id":2,"reuse_unbuilt_foundation":false},{"action":"if","when":{"resource_equals":{"category":"wood","amount":0}},"then":[{"action":"delete_unit","eligible":["build"]}]}`)
	view := checkedView("Foundations", "Buildings", "BuildCost", "BuildingAvailable", "IdleUnits", "EligibleUnits", "NearestBuildSite")
	view.units[0].Kind, view.units[0].Type = ai.UnitEconomic, ai.UnitTypeEconomic
	view.resources[ai.ResourceWood] = 0
	place := ai.ZoneRef{Space: ai.Coordinate{X: 2}}
	view.foundations = []ai.FoundationView{{BuildingID: 2, Planned: true, Location: place}, {BuildingID: 2, Location: ai.ZoneRef{Space: ai.Coordinate{X: 4}}}, {BuildingID: 2, Planned: true, Builders: 1, Location: ai.ZoneRef{Space: ai.Coordinate{X: 5}}}, {BuildingID: 3, Planned: true, Location: ai.ZoneRef{Space: ai.Coordinate{X: 6}}}}
	assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: 18, Item: 2, Place: place}, orderCall{Kind: 3, Subject: 91, Item: 2, Place: place, Clear: true}, orderCall{Kind: 12, Subject: 91, Clear: true})
}

func TestAttackResumesOriginalJob(t *testing.T) {
	for _, c := range []struct {
		kind ai.OrderKind
		call orderCall
	}{{ai.OrderKindGather, orderCall{Kind: 2, Subject: 91, Target: 401, Place: ai.ZoneRef{Space: ai.Coordinate{X: 3}}}}, {ai.OrderKindRepair, orderCall{Kind: 4, Subject: 91, Target: 800, Place: ai.ZoneRef{Zone: ai.Coordinate{X: 1}}}}, {ai.OrderKindRenew, orderCall{Kind: 5, Subject: 91, Target: 800, Place: ai.ZoneRef{Zone: ai.Coordinate{X: 1}}}}, {ai.OrderKindBuild, orderCall{Kind: 3, Subject: 91, Item: 2, Place: ai.ZoneRef{Space: ai.Coordinate{X: 2}}}}} {
		for _, state := range []string{"planned", "materialized", "missing"} {
			view := checkedView("EligibleUnits", "VisibleEnemyUnits", "Foundations", "Buildings")
			place := ai.ZoneRef{Space: ai.Coordinate{X: 2}}
			job := &ai.CurrentOrder{Kind: c.kind}
			switch c.kind {
			case ai.OrderKindGather:
				job.TargetResource = &view.knownResources[0]
			case ai.OrderKindRepair, ai.OrderKindRenew:
				job.TargetBuilding = &view.buildings[0]
			case ai.OrderKindBuild:
				job.TargetZone = &place
			}
			view.units[0].CurrentOrder = job
			view.foundations = []ai.FoundationView{{BuildingID: 2, Location: place, Planned: true}}
			if state != "planned" {
				view.foundations = nil
				if state == "materialized" {
					view.buildings = append(view.buildings, ai.BuildingView{InternalID: 801, BuildingID: 2, Location: place, UnderConstruction: true})
				}
			}
			model := actionsModel(t, `{"action":"attack","targets":"enemy_units","eligible":["gather","repair","renew","build"],"resume":true}`)
			expected := []namedCall{{"interrupt for attack", orderCall{Kind: 6, Subject: 91, Target: 201, Place: ai.ZoneRef{Space: ai.Coordinate{X: 4}}, Clear: true}}}
			if c.kind != ai.OrderKindBuild || state != "missing" {
				expected = append(expected, namedCall{"resume valid job", c.call})
			}
			assertScenarioCalls(t, state, model.DecideOrders(view), view, expected)
		}
	}
}

var spreadCapacityCases = []struct {
	options  string
	subjects []uint64
}{
	{`"overflow":"none"`, []uint64{91, 92}}, {`"capacity":0,"overflow":"none"`, nil},
	{`"capacity":1,"overflow":"none"`, []uint64{91}}, {`"capacity":2,"overflow":"none"`, []uint64{91, 92}},
	{`"capacity":0.5,"overflow":"none"`, []uint64{91}}, {`"capacity":"1 + 1","overflow":"none"`, []uint64{91, 92}},
	{`"capacity":0,"overflow":"round_robin"`, []uint64{91, 92}}, {`"capacity":1,"weight":0,"overflow":"none"`, []uint64{91, 92}},
	{`"capacity":1,"weight":-1,"overflow":"none"`, []uint64{91, 92}}, {`"capacity":1,"weight":2,"overflow":"none"`, []uint64{91}},
}

func TestSpreadCapacityAndWeight(t *testing.T) {
	for _, c := range spreadCapacityCases {
		model := actionsModel(t, `{"action":"attack","targets":"enemy_units","spread":{`+c.options+`}}`)
		view := checkedView("IdleUnits", "EligibleUnits", "VisibleEnemyUnits", "AssignedTo")
		second := view.units[0]
		second.InternalID = 92
		second.Location.Zone.X = 1
		view.units = []ai.UnitView{second, view.units[0]}
		var calls []orderCall
		for _, id := range c.subjects {
			calls = append(calls, orderCall{Kind: 6, Subject: id, Target: 201, Place: ai.ZoneRef{Space: ai.Coordinate{X: 4}}, Clear: true})
		}
		assertCalls(t, model.DecideOrders(view), view, calls...)
	}
}

func TestSpreadQueuedRounds(t *testing.T) {
	model := actionsModel(t, `{"action":"attack","targets":"enemy_units","spread":{"capacity":1,"queue":1,"overflow":"none"}}`)
	view := checkedView("IdleUnits", "EligibleUnits", "VisibleEnemyUnits", "AssignedTo")
	second := view.visibleEnemyUnits[0]
	second.InternalID = 202
	second.Location.Space.X = 5
	view.visibleEnemyUnits = append(view.visibleEnemyUnits, second)
	assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: 6, Subject: 91, Target: 201, Place: ai.ZoneRef{Space: ai.Coordinate{X: 4}}, Clear: true}, orderCall{Kind: 6, Subject: 91, Target: 202, Place: ai.ZoneRef{Space: ai.Coordinate{X: 5}}, Clear: false})
	if view.AssignedTo(false, 201, nil) != 1 || view.AssignedTo(false, 202, nil) != 1 {
		t.Fatal("queued attacks did not retain both assignments")
	}
}

