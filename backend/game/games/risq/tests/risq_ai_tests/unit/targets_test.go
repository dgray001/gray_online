package unit

import (
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"testing"
)

func TestTargetSetSelections(t *testing.T) {
	for _, c := range targetCases {
		for _, scenario := range []string{"selected", "below_min", "empty_subjects"} {
			t.Run(c.options+"/"+scenario, func(t *testing.T) {
				view := checkedView(append(c.queries, "IdleUnits", "EligibleUnits")...)
				count := 1
				view.allSpaces = []ai.SpaceInfo{{Space: ai.Coordinate{X: 2}, UnitCount: &count}}
				view.unexplored = []ai.ZoneRef{{Space: ai.Coordinate{X: 2}}}
				view.closestSpaces = func(from ai.Coordinate, where ai.SpaceCondition, excluded bool) []ai.Coordinate {
					if from != (ai.Coordinate{}) || excluded {
						t.Fatal("incorrect closest-space arguments")
					}
					return []ai.Coordinate{{X: 2}}
				}
				minimum := "0"
				if scenario == "below_min" {
					minimum = "1"
				}
				if scenario == "empty_subjects" {
					view.units = nil
				}
				model := actionsModel(t, `{"action":"move",`+c.options+`,"score":0,"min_score":`+minimum+`}`)
				var calls []orderCall
				if scenario == "selected" {
					calls = append(calls, orderCall{Kind: 1, Subject: 91, Place: c.place, Clear: true})
				}
				assertCalls(t, model.DecideOrders(view), view, calls...)
			})
		}
	}
}

func TestPickerAttackOrderModes(t *testing.T) {
	for _, c := range []struct {
		targets, mode string
		kind          uint8
		id            uint64
		x             int
	}{{"enemy_units", "unit", 6, 201, 4}, {"enemy_units", "zone", 9, 0, 4}, {"enemy_units", "space", 8, 0, 4}, {"enemy_buildings", "unit", 7, 301, 5}, {"enemy_buildings", "zone", 9, 0, 5}, {"enemy_buildings", "space", 8, 0, 5}} {
		model := actionsModel(t, `{"action":"attack","targets":"`+c.targets+`","order":"`+c.mode+`"}`)
		view := checkedView("IdleUnits", "EligibleUnits", "VisibleEnemyUnits", "KnownEnemyBuildings")
		assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: c.kind, Subject: 91, Target: c.id, Place: ai.ZoneRef{Space: ai.Coordinate{X: c.x}}, Clear: true})
	}
}

var targetCases = []struct {
	options string
	place   ai.ZoneRef
	queries []string
}{
	{`"targets":"enemy_units"`, ai.ZoneRef{Space: ai.Coordinate{X: 4}}, []string{"VisibleEnemyUnits"}},
	{`"targets":"enemy_buildings"`, ai.ZoneRef{Space: ai.Coordinate{X: 5}}, []string{"KnownEnemyBuildings"}},
	{`"targets":"known_resources","category":"wood"`, ai.ZoneRef{Space: ai.Coordinate{X: 3}}, []string{"KnownResources"}},
	{`"targets":"own_buildings"`, ai.ZoneRef{Zone: ai.Coordinate{X: 1}}, []string{"Buildings"}},
	{`"targets":"home"`, ai.ZoneRef{Zone: ai.Coordinate{X: 1}}, []string{"Buildings"}},
	{`"targets":"retreat","to":"home"`, ai.ZoneRef{Zone: ai.Coordinate{X: 1}}, []string{"Buildings"}},
	{`"targets":"unexplored"`, ai.ZoneRef{Space: ai.Coordinate{X: 2}}, []string{"NearestUnexplored"}},
	{`"targets":"unidentified_units"`, ai.ZoneRef{Space: ai.Coordinate{X: 2}}, []string{"AllSpaces"}},
	{`"targets":"closest_spaces"`, ai.ZoneRef{Space: ai.Coordinate{X: 2}}, []string{"ClosestSpaces"}},
	{`"targets":"coordinate","coordinate":{"x":"-1 - 1","y":1}`, ai.ZoneRef{Space: ai.Coordinate{X: -2, Y: 1}}, nil},
}
