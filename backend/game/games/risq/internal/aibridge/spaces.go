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
	count := 0
	for _, unit := range v.units {
		if unit.GarrisonedIn == nil && unit.Space == space.Coordinate {
			count++
		}
	}
	return count, true
}

func (v *aiView) AllZones() []ai.ZoneInfo {
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

func (v *aiView) OwnedSpaces() int {
	owned := 0
	for _, space := range v.spaces {
		if owner, known := spaceOwner(space); known && owner == v.playerId() {
			owned++
		}
	}
	return owned
}

func (v *aiView) AllSpaces() []ai.SpaceInfo {
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

func (v *aiView) EnemyDistance(from ai.Coordinate) int {
	if !v.enemy_spaces_known {
		seen := make(map[ai.Coordinate]bool)
		for _, b := range v.KnownEnemyBuildings() {
			if space := b.Location.Space; !seen[space] {
				seen[space] = true
				v.enemy_spaces = append(v.enemy_spaces, space)
			}
		}
		v.enemy_spaces_known = true
	}
	best := -1
	for _, space := range v.enemy_spaces {
		if d := v.SpaceDistance(from, space); best == -1 || d < best {
			best = d
		}
	}
	return best
}

func (v *aiView) RegionProgress(space ai.Coordinate) float64 {
	if owner, known := spaceOwner(v.spaces[space]); known && owner == v.playerId() {
		return 0
	}
	if v.space_by_key == nil {
		v.space_by_key = make(map[uint]*snapSpace, len(v.spaces))
		for coordinate, s := range v.spaces {
			v.space_by_key[spaceKey(coordinate)] = s
		}
	}
	key := spaceKey(space)
	for _, region := range v.game.Regions {
		member, owned := false, 0
		for _, k := range region.Spaces {
			member = member || k == key
			if owner, known := spaceOwner(v.space_by_key[k]); known && owner == v.playerId() {
				owned++
			}
		}
		if member {
			return float64(owned+1) / float64(len(region.Spaces))
		}
	}
	return 0
}

func (v *aiView) OwnBuildingsIn(space ai.Coordinate) int {
	count := 0
	for _, b := range v.me.Buildings {
		if toCoordinate(b.Space) == space {
			count++
		}
	}
	for location, f := range v.foundationsByLocation() {
		if f.Planned && location.Space == space {
			count++
		}
	}
	return count
}

func (v *aiView) NearestUnexplored(from ai.ZoneRef) ([]ai.ZoneRef, bool) {
	var tied []ai.ZoneRef
	vision := uint8(defs.VisibilityUnexplored)
	for _, space := range v.ClosestSpaces(from.Space, ai.SpaceCondition{Vision: &vision}, false) {
		tied = append(tied, ai.ZoneRef{Space: space})
	}
	return tied, len(tied) > 0
}

func (v *aiView) ownSpaceCount() int {
	count := 0
	for _, s := range v.spaces {
		if owner, known := spaceOwner(s); known && owner == v.playerId() {
			count++
		}
	}
	return count
}

// Mirrors the build rule against snapshot ownership: a space I own, an unowned one bordering one I own, or any unowned space if owning 0
func (v *aiView) buildable(space *snapSpace) bool {
	owner, known := spaceOwner(space)
	if !known || owner != -1 {
		return known && owner == v.playerId()
	}
	if v.ownSpaceCount() == 0 {
		return true
	}
	for _, adjacent := range v.SpaceNeighbors(toCoordinate(space.Coordinate)) {
		if adj_owner, adj_known := spaceOwner(v.spaces[adjacent]); adj_known && adj_owner == v.playerId() {
			return true
		}
	}
	return false
}

const outpostBuildingID = 21

// Every zone the building can go on. Outposts exist to claim land, so they only go in unowned spaces with no other foundation
func (v *aiView) BuildSites(building_id uint32, include func(ai.Coordinate) bool) []ai.ZoneRef {
	foundations := v.foundationsByLocation()
	claimed := make(map[ai.Coordinate]bool, len(foundations))
	for location := range foundations {
		claimed[location.Space] = true
	}
	sites := make([]ai.ZoneRef, 0)
	wall := v.config.Buildings[building_id].Is_wall
	for _, entry := range v.zones {
		if include != nil && !include(entry.ref.Space) {
			continue
		}
		if wall && entry.ref.Zone == (ai.Coordinate{}) {
			continue
		}
		if entry.zone.Resource != nil || entry.zone.Building != nil || foundations[entry.ref] != nil || !v.buildable(entry.space) {
			continue
		}
		if owner, known := spaceOwner(entry.space); building_id == outpostBuildingID && (!known || owner != -1 || claimed[entry.ref.Space]) {
			continue
		}
		sites = append(sites, entry.ref)
	}
	return sites
}

func (v *aiView) NearestBuildSite(from ai.ZoneRef, building_id uint32, include func(ai.Coordinate) bool) (ai.ZoneRef, bool) {
	var best ai.ZoneRef
	found, best_distance := false, -1
	for _, site := range v.BuildSites(building_id, include) {
		d := v.LocationDistance(from, site)
		if !found || d < best_distance || (d == best_distance && zoneKey(site) < zoneKey(best)) {
			best, best_distance, found = site, d, true
		}
	}
	return best, found
}
