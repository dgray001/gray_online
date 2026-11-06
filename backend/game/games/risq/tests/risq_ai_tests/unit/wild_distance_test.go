package unit

import (
	"testing"

	. "github.com/dgray001/gray_online/game/games/risq/ai"
)

func TestDistanceInAggregate(t *testing.T) {
	fixture := loopFixture()
	fixture.spaces = []SpaceInfo{
		{Space: Coordinate{X: 0, Y: 0}, Owner: 0},
		{Space: Coordinate{X: 2, Y: 0}},
	}
	fixture.resources = map[ResourceCategory][]ResourceView{
		ResourceFood: {
			{Location: ZoneRef{Space: Coordinate{X: 2, Y: 0}}, AmountLeft: 300, GatherCapacity: 4, Category: ResourceFood},
		},
	}
	config := rule(`
		{"action":"select","pick":{"kind":"spaces","where":{"if":"var(target.x) == 0"}},"into":{"set":"home"}},
		{"action":"for_each","select":{"kind":"spaces","where":{"in_set":"home"},"limit":1},"as":"home_space","do":[
			{"action":"set_var","name":"wild_food_slots","value":{"sum":{"kind":"resources","where":{"category":"food"},"of":"min(var(item.gather_capacity), var(item.amount_left) / 36) / (1 + distance(home_space, item))"}}}
		]}
	`)
	view := selectTestView{loopTestView: fixture, idle: fixture.units, turn: 1}
	res := decide(t, config, view, "wild_food_slots")
	t.Logf("wild_food_slots = %v", res["wild_food_slots"])
	if res["wild_food_slots"] == 0 {
		t.Fatalf("expected non-zero wild_food_slots")
	}
}
