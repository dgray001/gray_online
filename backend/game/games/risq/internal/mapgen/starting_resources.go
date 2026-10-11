package mapgen

import (
	"encoding/json"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"strconv"
)

func defaultStartingResources(level defs.MapResourceLevel) []playerStartResourceJSON {
	var home []playerStartResourceJSON
	switch level {
	case defs.MapResourceLevel_SPARSE:
		home = homeStartingResources(uniformTreeIds, uniformTreeIds, 0)
	case defs.MapResourceLevel_RICH:
		home = homeStartingResources(uniformGroveIds, uniformGroveIds, 42)
	case defs.MapResourceLevel_ABUNDANT:
		home = homeStartingResources(uniformForestIds, uniformGroveIds, 43)
	default:
		home = homeStartingResources(uniformGroveIds, uniformTreeIds, 41)
	}
	return append(home, outerStartingResources(level)...)
}

func homeStartingResources(firstWoodIds []uint32, secondWoodIds []uint32, stoneId uint32) []playerStartResourceJSON {
	count := ScriptExpr{raw: json.RawMessage("1")}
	resources := []playerStartResourceJSON{
		{ResourceIds: firstWoodIds, Count: count, Distance: []int{0, 0}, Zone: "edge"},
		{ResourceIds: secondWoodIds, Count: count, Distance: []int{0, 0}, Zone: "edge"},
		{ResourceId: 2, Count: count, Distance: []int{0, 0}, Zone: "edge"},
		{ResourceId: 1, Count: count, Distance: []int{0, 0}, Zone: "edge"},
	}
	if stoneId != 0 {
		resources = append(resources, playerStartResourceJSON{ResourceId: stoneId, Count: count, Distance: []int{0, 0}, Zone: "edge"})
	}
	return resources
}

func outerStartingResources(level defs.MapResourceLevel) []playerStartResourceJSON {
	count := func(base float64) ScriptExpr {
		return ScriptExpr{raw: json.RawMessage(strconv.FormatFloat(base, 'f', -1, 64))}
	}
	switch level {
	case defs.MapResourceLevel_ABUNDANT:
		return []playerStartResourceJSON{
			{ResourceIds: uniformGroveIds, Count: count(1), Distance: []int{1, 1}},
			{ResourceIds: uniformForestIds, Count: count(7), Distance: []int{1, 1}},
			{ResourceId: 2, Count: count(4), Distance: []int{1, 1}},
			{ResourceId: 1, Count: count(4), Distance: []int{1, 1}},
			{ResourceId: 42, Count: count(2), Distance: []int{1, 1}},
			{ResourceId: 43, Count: count(2), Distance: []int{1, 1}},
		}
	case defs.MapResourceLevel_RICH:
		return []playerStartResourceJSON{
			{ResourceIds: uniformGroveIds, Count: count(2), Distance: []int{1, 1}},
			{ResourceIds: uniformForestIds, Count: count(5), Distance: []int{1, 1}},
			{ResourceId: 2, Count: count(3), Distance: []int{1, 1}},
			{ResourceId: 1, Count: count(3), Distance: []int{1, 1}},
			{ResourceId: 41, Count: count(2), Distance: []int{1, 1}},
			{ResourceId: 42, Count: count(2), Distance: []int{1, 1}},
			{ResourceId: 43, Count: count(1), Distance: []int{1, 1}},
		}
	case defs.MapResourceLevel_SPARSE:
		return []playerStartResourceJSON{
			{ResourceIds: uniformForestIds, Count: count(2), Distance: []int{1, 1}},
			{ResourceIds: uniformGroveIds, Count: count(2), Distance: []int{1, 1}},
			{ResourceIds: uniformTreeIds, Count: count(2), Distance: []int{1, 1}},
			{ResourceId: 41, Count: count(1), Distance: []int{1, 1}},
			{ResourceId: 42, Count: count(1), Distance: []int{1, 1}},
			{ResourceId: 2, Count: count(1), Distance: []int{1, 1}},
			{ResourceId: 1, Count: count(1), Distance: []int{1, 1}},
		}
	}
	return []playerStartResourceJSON{
		{ResourceIds: uniformGroveIds, Count: count(3), Distance: []int{1, 1}},
		{ResourceIds: uniformForestIds, Count: count(3), Distance: []int{1, 1}},
		{ResourceId: 2, Count: count(2), Distance: []int{1, 1}},
		{ResourceId: 1, Count: count(2), Distance: []int{1, 1}},
		{ResourceId: 42, Count: count(1), Distance: []int{1, 1}},
		{ResourceId: 43, Count: count(1), Distance: []int{1, 1}},
	}
}
