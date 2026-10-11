package unit

import (
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"testing"
)

func TestFoundationRefundAndReplacement(t *testing.T) {
	model := actionsModel(t, `{"action":"cancel_foundations","building_ids":[2]},{"action":"cancel_foundations","building_ids":[2]},{"action":"build","building_id":2,"reuse_unbuilt_foundation":false},{"action":"if","when":{"resource_equals":{"category":"wood","amount":0}},"then":[{"action":"delete_unit","eligible":["build"]}]}`)
	view := checkedView("Foundations", "Buildings", "BuildCost", "BuildingAvailable", "IdleUnits", "EligibleUnits", "NearestBuildSite")
	view.units[0].Kind, view.units[0].Type = ai.UnitEconomic, ai.UnitTypeEconomic
	view.resources[ai.ResourceWood] = 0
	place := ai.ZoneRef{Space: ai.Coordinate{X: 2}}
	view.foundations = []ai.FoundationView{{BuildingID: 2, Planned: true, Location: place}, {BuildingID: 2, Location: ai.ZoneRef{Space: ai.Coordinate{X: 4}}}, {BuildingID: 2, Planned: true, Builders: 1, Location: ai.ZoneRef{Space: ai.Coordinate{X: 5}}}, {BuildingID: 3, Planned: true, Location: ai.ZoneRef{Space: ai.Coordinate{X: 6}}}}
	assertCalls(t, model.DecideOrders(view), view, orderCall{Kind: 18, Item: 2, Place: place}, orderCall{Kind: 3, Subject: 91, Item: 2, Place: place, Clear: true}, orderCall{Kind: 12, Subject: 91, Clear: true})
}
