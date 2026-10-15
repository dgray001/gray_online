package mapgen

import (
	"encoding/json"
	"fmt"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"slices"
	"strconv"
)

// Swaps the default tree, grove (+10) and forest (+20) pools for the given species ids
func withTreeSpecies(ctx *mapScriptContext, resources []playerStartResourceJSON, species []ScriptExpr) ([]playerStartResourceJSON, error) {
	trees := make([]uint32, len(species))
	for i, expr := range species {
		id, err := expr.resolveInt(ctx.vars)
		if err != nil {
			return nil, err
		}
		if _, ok := ctx.config().Resources[uint32(id)]; !ok {
			return nil, fmt.Errorf("player_starts tree_ids: unknown resource id %d", id)
		}
		trees[i] = uint32(id)
	}
	shifted := func(offset uint32) []uint32 {
		ids := make([]uint32, len(trees))
		for i, id := range trees {
			ids[i] = id + offset
		}
		return ids
	}
	for i, r := range resources {
		switch {
		case slices.Equal(r.ResourceIds, uniformTreeIds):
			resources[i].ResourceIds = trees
		case slices.Equal(r.ResourceIds, uniformGroveIds):
			resources[i].ResourceIds = shifted(10)
		case slices.Equal(r.ResourceIds, uniformForestIds):
			resources[i].ResourceIds = shifted(20)
		}
	}
	return resources, nil
}

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
