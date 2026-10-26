package mapgen

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
)

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
			resources[i].ResourceIds = shifted(100)
		case slices.Equal(r.ResourceIds, uniformForestIds):
			resources[i].ResourceIds = shifted(200)
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
		home = homeStartingResources(uniformGroveIds, uniformGroveIds, 102)
	case defs.MapResourceLevel_ABUNDANT:
		home = homeStartingResources(uniformForestIds, uniformGroveIds, 103)
	default:
		home = homeStartingResources(uniformGroveIds, uniformTreeIds, 101)
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
			{ResourceId: 102, Count: count(2), Distance: []int{1, 1}},
			{ResourceId: 103, Count: count(2), Distance: []int{1, 1}},
		}
	case defs.MapResourceLevel_RICH:
		return []playerStartResourceJSON{
			{ResourceIds: uniformGroveIds, Count: count(2), Distance: []int{1, 1}},
			{ResourceIds: uniformForestIds, Count: count(5), Distance: []int{1, 1}},
			{ResourceId: 2, Count: count(3), Distance: []int{1, 1}},
			{ResourceId: 1, Count: count(3), Distance: []int{1, 1}},
			{ResourceId: 101, Count: count(2), Distance: []int{1, 1}},
			{ResourceId: 102, Count: count(2), Distance: []int{1, 1}},
			{ResourceId: 103, Count: count(1), Distance: []int{1, 1}},
		}
	case defs.MapResourceLevel_SPARSE:
		return []playerStartResourceJSON{
			{ResourceIds: uniformForestIds, Count: count(2), Distance: []int{1, 1}},
			{ResourceIds: uniformGroveIds, Count: count(2), Distance: []int{1, 1}},
			{ResourceIds: uniformTreeIds, Count: count(2), Distance: []int{1, 1}},
			{ResourceId: 101, Count: count(1), Distance: []int{1, 1}},
			{ResourceId: 102, Count: count(1), Distance: []int{1, 1}},
			{ResourceId: 2, Count: count(1), Distance: []int{1, 1}},
			{ResourceId: 1, Count: count(1), Distance: []int{1, 1}},
		}
	}
	return []playerStartResourceJSON{
		{ResourceIds: uniformGroveIds, Count: count(3), Distance: []int{1, 1}},
		{ResourceIds: uniformForestIds, Count: count(3), Distance: []int{1, 1}},
		{ResourceId: 2, Count: count(2), Distance: []int{1, 1}},
		{ResourceId: 1, Count: count(2), Distance: []int{1, 1}},
		{ResourceId: 102, Count: count(1), Distance: []int{1, 1}},
		{ResourceId: 103, Count: count(1), Distance: []int{1, 1}},
	}
}

func planStartResources(ctx *mapScriptContext, resources []playerStartResourceJSON, area_size int) ([]startResourceSlot, error) {
	used := make(map[[4]int]bool)
	slots := make([]startResourceSlot, 0)
	for _, r := range resources {
		ids, err := r.resourceIds(ctx.config())
		if err != nil {
			return nil, err
		}
		first_slot := len(slots)
		count, err := r.Count.resolveInt(ctx.vars)
		if err != nil {
			return nil, err
		}
		if !validateResourceZone(r.Zone) {
			return nil, fmt.Errorf("resource %d: invalid zone %q", r.ResourceId, r.Zone)
		}
		lo, hi := 0, area_size
		if len(r.Distance) == 2 {
			lo, hi = r.Distance[0], min(r.Distance[1], area_size)
		} else if len(r.Distance) != 0 {
			return nil, fmt.Errorf("resource %d distance must be [min, max]", r.ResourceId)
		}
		candidates := make([][4]int, 0)
		for dq := -hi; dq <= hi; dq++ {
			for dr := -hi; dr <= hi; dr++ {
				if d := int(game_utils.AxialDistance(game_utils.Coordinate2D{}, game_utils.Coordinate2D{X: dq, Y: dr})); d < lo || d > hi {
					continue
				}
				if resourceLocalMatches(game_utils.Coordinate2D{}, r.Zone) {
					candidates = append(candidates, [4]int{dq, dr, 0, 0})
				}
				for _, z := range game_utils.AxialDirectionVectors() {
					if resourceLocalMatches(z, r.Zone) {
						candidates = append(candidates, [4]int{dq, dr, z.X, z.Y})
					}
				}
			}
		}
		candidates = util.ShuffleFrom(ctx.rng, candidates)
		placed := 0
		for _, c := range candidates {
			if placed >= count {
				break
			}
			if used[c] {
				continue
			}
			used[c] = true
			placed++
			slots = append(slots, startResourceSlot{resource_id: r.ResourceId, min_distance: lo, max_distance: hi, zone_kind: r.Zone,
				space_offset: game_utils.Coordinate2D{X: c[0], Y: c[1]}, zone: game_utils.Coordinate2D{X: c[2], Y: c[3]}})
		}
		for ; placed < count; placed++ {
			slots = append(slots, startResourceSlot{resource_id: r.ResourceId, min_distance: lo, max_distance: hi, zone_kind: r.Zone, unplanned: true})
		}
		for i := first_slot; i < len(slots); i++ {
			slots[i].resource_id = ids[0]
			if len(ids) > 1 {
				slots[i].resource_id = ids[ctx.rng.Intn(len(ids))]
			}
		}
	}
	return slots, nil
}

// The free zone nearest from whose space is within [lo, hi] spaces of it, relaxing the selector before giving up
func nearestFreeZone(zones []Zone, from game_utils.Coordinate2D, lo int, hi int, selector string) Zone {
	if z := nearestSelectedFreeZone(zones, from, lo, hi, selector); z != nil || selector == "" {
		return z
	}
	return nearestSelectedFreeZone(zones, from, lo, hi, "")
}

// The first of any distance tie in zones' order wins
func nearestSelectedFreeZone(zones []Zone, from game_utils.Coordinate2D, lo int, hi int, selector string) Zone {
	var nearest Zone
	nearest_distance := -1
	for _, z := range zones {
		d := int(game_utils.AxialDistance(from, z.Space().Coordinate()))
		if z.Occupied() || !resourceZoneMatches(z, selector) || d < lo || d > hi || (nearest_distance != -1 && d >= nearest_distance) {
			continue
		}
		nearest, nearest_distance = z, d
	}
	return nearest
}

func plannedResourceZone(ctx *mapScriptContext, home game_utils.Coordinate2D, rotation int, slot startResourceSlot, in_area map[uint]bool) Zone {
	offset := rotateAxial(slot.space_offset, rotation)
	s := ctx.board.Space(game_utils.Coordinate2D{X: home.X + offset.X, Y: home.Y + offset.Y})
	if slot.unplanned || s == nil || !in_area[s.Key()] {
		return nil
	}
	if z := s.Zone(rotateAxial(slot.zone, rotation)); z != nil && !z.Occupied() && resourceZoneMatches(z, slot.zone_kind) {
		return z
	}
	return nil
}

func startResourceZone(ctx *mapScriptContext, home game_utils.Coordinate2D, rotation int, slot startResourceSlot, in_area map[uint]bool, area_zones []Zone, board_zones []Zone) Zone {
	if z := plannedResourceZone(ctx, home, rotation, slot, in_area); z != nil {
		return z
	}
	if z := nearestFreeZone(area_zones, home, slot.min_distance, slot.max_distance, slot.zone_kind); z != nil {
		return z
	}
	if z := nearestFreeZone(area_zones, home, 0, math.MaxInt, slot.zone_kind); z != nil {
		return z
	}
	return nearestFreeZone(board_zones, home, 0, math.MaxInt, slot.zone_kind)
}

// The mirrored zone when it is free, else the free zone nearest it (in the player's area first)
func mirroredResourceZone(ctx *mapScriptContext, mirrored game_utils.Coordinate2D, rotation int, slot startResourceSlot, area_zones []Zone, board_zones []Zone) Zone {
	if s := ctx.board.Space(mirrored); ctx.startAllowed(s) {
		if z := s.Zone(rotateAxial(slot.zone, rotation)); z != nil && !z.Occupied() {
			return z
		}
	}
	if z := nearestFreeZone(area_zones, mirrored, 0, math.MaxInt, slot.zone_kind); z != nil {
		return z
	}
	return nearestFreeZone(board_zones, mirrored, 0, math.MaxInt, slot.zone_kind)
}

// Places every start resource for every player, skipping any that find no free zone left
func placeStartResources(ctx *mapScriptContext, starts []playerStartInfo, footprints [][]Space, slots []startResourceSlot) error {
	board_zones := ctx.startAllowedZones()
	for i, start := range starts {
		in_area := make(map[uint]bool, len(footprints[i]))
		area_zones := make([]Zone, 0)
		for _, s := range footprints[i] {
			in_area[s.Key()] = true
			area_zones = append(area_zones, s.Zones()...)
		}
		area_zones = util.ShuffleFrom(ctx.rng, area_zones)
		rotation := directionIndex(start.direction) - directionIndex(starts[0].direction)
		for slot_index := range slots {
			slot := &slots[slot_index]
			zone := startResourceZone(ctx, start.space.Coordinate(), rotation, *slot, in_area, area_zones, board_zones)
			if zone == nil {
				continue
			}
			if i == 0 {
				slot.space_offset = game_utils.Coordinate2D{X: zone.Space().Coordinate().X - start.space.Coordinate().X, Y: zone.Space().Coordinate().Y - start.space.Coordinate().Y}
				slot.zone = zone.Local()
				slot.unplanned = false
			} else {
				rotated := rotateAxial(slot.space_offset, rotation)
				mirrored := game_utils.Coordinate2D{X: start.space.Coordinate().X + rotated.X, Y: start.space.Coordinate().Y + rotated.Y}
				if zone = mirroredResourceZone(ctx, mirrored, rotation, *slot, area_zones, board_zones); zone == nil {
					continue
				}
			}
			ctx.board.PlaceResource(zone, slot.resource_id)
		}
	}
	return nil
}
