package unit

import (
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"testing"
)

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
