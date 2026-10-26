package aibridge

import (
	"slices"

	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

func (v *aiView) buildMatchingSpaces(condition ai.SpaceCondition) []ai.Coordinate {
	var matches []ai.Coordinate
	for _, row := range v.game.Spaces {
		for _, space := range row {
			if space != nil && v.spaceMatches(space, condition) {
				matches = append(matches, toCoordinate(space.Coordinate))
			}
		}
	}
	return matches
}

func (v *aiView) MatchingSpaces(condition ai.SpaceCondition) []ai.Coordinate {
	return v.matchingSpaces(condition)
}

func (v *aiView) spaceMatches(space *snapSpace, c ai.SpaceCondition) bool {
	if !spaceTerrainMatches(space, c) {
		return false
	}
	if c.UnitCountAtLeast != nil || c.UnitCountAtMost != nil {
		count, known := v.spaceUnitCount(space)
		if !known || c.UnitCountAtLeast != nil && count < *c.UnitCountAtLeast || c.UnitCountAtMost != nil && count > *c.UnitCountAtMost {
			return false
		}
	}
	if len(c.ResourceIDs) > 0 && !spaceHasResource(space, c.ResourceIDs) {
		return false
	}
	if (len(c.BuildingIDs) > 0 || c.BuildingOwner != "" || c.BuildingPlayerID != nil) && !v.spaceHasBuilding(space, c) {
		return false
	}
	if c.Vision != nil && space.Visibility != *c.Vision || c.VisionAtLeast != nil && space.Visibility < *c.VisionAtLeast || c.VisionAtMost != nil && space.Visibility > *c.VisionAtMost {
		return false
	}
	if c.Owner != "" && c.Owner != "any" || c.PlayerID != nil {
		owner, known := spaceOwner(space)
		if !known || !v.ownerMatches(owner, c.Owner, c.PlayerID) {
			return false
		}
	}
	for _, child := range c.All {
		if !v.spaceMatches(space, child) {
			return false
		}
	}
	if c.Any != nil {
		matches := false
		for _, child := range c.Any {
			matches = matches || v.spaceMatches(space, child)
		}
		if !matches {
			return false
		}
	}
	if c.Not != nil && v.spaceMatches(space, *c.Not) {
		return false
	}
	return true
}

func spaceTerrainMatches(space *snapSpace, c ai.SpaceCondition) bool {
	t := space.TerrainStats
	filtered := len(c.TerrainTypes) > 0 || c.Passable != nil || c.IntraCostAtLeast != nil || c.IntraCostAtMost != nil || c.InterCostAtLeast != nil || c.InterCostAtMost != nil || c.BuildSpeedModifierAtLeast != nil || c.BuildSpeedModifierAtMost != nil
	if t == nil || space.Visibility == defs.VisibilityUnexplored {
		return !filtered
	}
	if len(c.TerrainTypes) > 0 {
		matched := false
		for _, name := range c.TerrainTypes {
			kind, err := defs.ParseTerrainType(name)
			matched = matched || err == nil && uint8(kind) == space.TerrainType
		}
		if !matched {
			return false
		}
	}
	return !(c.Passable != nil && *c.Passable == t.Impassable ||
		c.IntraCostAtLeast != nil && t.IntraCost < *c.IntraCostAtLeast || c.IntraCostAtMost != nil && t.IntraCost > *c.IntraCostAtMost ||
		c.InterCostAtLeast != nil && t.InterCost < *c.InterCostAtLeast || c.InterCostAtMost != nil && t.InterCost > *c.InterCostAtMost ||
		c.BuildSpeedModifierAtLeast != nil && t.BuildSpeedModifier < *c.BuildSpeedModifierAtLeast || c.BuildSpeedModifierAtMost != nil && t.BuildSpeedModifier > *c.BuildSpeedModifierAtMost)
}

func (v *aiView) ownerMatches(owner int, selector string, player_id *int) bool {
	if player_id != nil && owner != *player_id {
		return false
	}
	switch selector {
	case "own":
		return owner == v.playerId()
	case "enemy":
		return owner >= 0 && owner != v.playerId()
	case "unowned":
		return owner == -1
	}
	return true
}

func spaceHasResource(space *snapSpace, ids []uint32) bool {
	for _, row := range space.Zones {
		for _, zone := range row {
			if r := zone.Resource; r != nil && r.ResourcesLeft > 0 && slices.Contains(ids, r.ResourceId) {
				return true
			}
		}
	}
	return false
}

func (v *aiView) spaceHasBuilding(space *snapSpace, c ai.SpaceCondition) bool {
	for _, b := range space.Buildings {
		if (len(c.BuildingIDs) == 0 || slices.Contains(c.BuildingIDs, b.BuildingId)) && v.ownerMatches(b.PlayerId, c.BuildingOwner, c.BuildingPlayerID) {
			return true
		}
	}
	return false
}

func (v *aiView) spaceUnitCount(space *snapSpace) (int, bool) {
	if space.UnitCount != nil {
		return *space.UnitCount, true
	}
	if space.Visibility < defs.VisibilityGood {
		return 0, false
	}
	if v.space_unit_counts == nil {
		v.space_unit_counts = make(map[ai.Coordinate]int)
		for _, unit := range v.units {
			if unit.GarrisonedIn == nil {
				v.space_unit_counts[toCoordinate(unit.Space)]++
			}
		}
	}
	return v.space_unit_counts[toCoordinate(space.Coordinate)], true
}

func (v *aiView) buildAllZones() []ai.ZoneInfo {
	zones := make([]ai.ZoneInfo, 0, len(v.zones))
	for _, entry := range v.zones {
		info := ai.ZoneInfo{Location: entry.ref, HasResource: entry.zone.Resource != nil, BuildingPlayer: -1}
		if b := entry.zone.Building; b != nil {
			info.BuildingPlayer = b.PlayerId
		}
		zones = append(zones, info)
	}
	return zones
}

func (v *aiView) buildAllSpaces() []ai.SpaceInfo {
	infos := make([]ai.SpaceInfo, 0, len(v.spaces))
	for _, row := range v.game.Spaces {
		for _, space := range row {
			if space != nil {
				owner, _ := spaceOwner(space)
				info := ai.SpaceInfo{Space: toCoordinate(space.Coordinate), Vision: space.Visibility, Owner: owner, UnitCount: space.UnitCount, GoldIncome: space.GoldIncome}
				if terrain := space.TerrainStats; terrain != nil {
					info.TerrainType, info.Passable = space.TerrainType, !terrain.Impassable
					info.IntraCost, info.InterCost = terrain.IntraCost, terrain.InterCost
					info.BuildSpeedModifier = terrain.BuildSpeedModifier
				}
				infos = append(infos, info)
			}
		}
	}
	return infos
}

func (v *aiView) AllRegions() []ai.RegionInfo {
	byKey := make(map[uint]ai.Coordinate, len(v.spaces))
	for coordinate := range v.spaces {
		byKey[spaceKey(coordinate)] = coordinate
	}
	regions := make([]ai.RegionInfo, 0, len(v.game.Regions))
	for _, region := range v.game.Regions {
		info := ai.RegionInfo{Name: region.Name, GoldBonus: region.GoldBonus, Owner: region.Owner, HasUnexplored: region.HasUnexplored}
		for _, key := range region.Spaces {
			info.Spaces = append(info.Spaces, byKey[key])
		}
		regions = append(regions, info)
	}
	return regions
}
