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
