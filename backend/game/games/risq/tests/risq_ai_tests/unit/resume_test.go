package unit

import (
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"testing"
)

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
