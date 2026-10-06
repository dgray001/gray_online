package aibridge

import (
	"slices"

	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

func (v *aiView) matchingSpaces(condition ai.SpaceCondition) []ai.Coordinate {
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

func (v *aiView) CountSpaces(condition ai.SpaceCondition, include func(ai.Coordinate) bool) int {
	count := 0
	for _, space := range v.matchingSpaces(condition) {
		if include(space) {
			count++
		}
	}
	return count
}

func (v *aiView) ClosestSpaces(reference ai.Coordinate, condition ai.SpaceCondition, exclude_reference bool) []ai.Coordinate {
	var nearest []ai.Coordinate
	best_distance := -1
	for _, space := range v.matchingSpaces(condition) {
		if exclude_reference && space == reference {
			continue
		}
		distance := v.SpaceDistance(reference, space)
		if best_distance == -1 || distance < best_distance {
			nearest, best_distance = []ai.Coordinate{space}, distance
		} else if distance == best_distance {
			nearest = append(nearest, space)
		}
	}
	return nearest
}

func (v *aiView) spaceMatches(space *snapSpace, c ai.SpaceCondition) bool {
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
	count := 0
	for _, unit := range v.units {
		if unit.GarrisonedIn == nil && unit.Space == space.Coordinate {
			count++
		}
	}
	return count, true
}
