package risq

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

type categoryBest struct {
	units         map[defs.TargetCategory]*RisqUnit
	unit_dist     map[defs.TargetCategory]int
	building      *RisqBuilding
	building_dist int
}

func newCategoryBest() *categoryBest {
	return &categoryBest{units: make(map[defs.TargetCategory]*RisqUnit), unit_dist: make(map[defs.TargetCategory]int)}
}

func (c *categoryBest) considerUnit(target *RisqUnit, dist int) {
	cat := targetCategoryOf(target)
	if existing, ok := c.units[cat]; !ok || dist < c.unit_dist[cat] || (dist == c.unit_dist[cat] && target.internal_id < existing.internal_id) {
		c.units[cat] = target
		c.unit_dist[cat] = dist
	}
}

func (c *categoryBest) considerBuilding(target *RisqBuilding, dist int) {
	if c.building == nil || dist < c.building_dist || (dist == c.building_dist && target.internal_id < c.building.internal_id) {
		c.building = target
		c.building_dist = dist
	}
}

func (c *categoryBest) pick(priority []defs.TargetCategory) Attackable {
	ranked := make(map[defs.TargetCategory]bool, len(priority))
	for _, cat := range priority {
		if cat == defs.TargetCategory_NONE || ranked[cat] {
			continue
		}
		ranked[cat] = true
		if cat == defs.TargetCategory_BUILDING {
			if c.building != nil {
				return c.building
			}
		} else if target, ok := c.units[cat]; ok {
			return target
		}
	}
	var best_unit *RisqUnit
	best_dist := -1
	for cat, target := range c.units {
		if ranked[cat] {
			continue
		}
		if best_unit == nil || c.unit_dist[cat] < best_dist || (c.unit_dist[cat] == best_dist && target.internal_id < best_unit.internal_id) {
			best_unit, best_dist = target, c.unit_dist[cat]
		}
	}
	if !ranked[defs.TargetCategory_BUILDING] && c.building != nil && (best_unit == nil || c.building_dist < best_dist) {
		return c.building
	}
	if best_unit != nil {
		return best_unit
	}
	return nil
}

// attackOrderType picks the unit- or building-attack order type variant for target.
func attackOrderType(target Attackable, unit_type defs.OrderType, building_type defs.OrderType) defs.OrderType {
	if target.OrderableType() == defs.OrderableType_UNIT {
		return unit_type
	}
	return building_type
}

func zoneAttackTarget(zone *RisqZone, player_id int, priority []defs.TargetCategory) Attackable {
	best := newCategoryBest()
	for _, target := range zone.units {
		if target.deleted || target.player_id == player_id {
			continue
		}
		best.considerUnit(target, 0)
	}
	if target := zoneEnemyBuilding(zone, player_id); target != nil {
		best.considerBuilding(target, 0)
	}
	return best.pick(priority)
}

func spaceAttackTarget(u *RisqUnit, space *RisqSpace) Attackable {
	near, far := newCategoryBest(), newCategoryBest()
	for _, row := range space.zones {
		for _, zone := range row {
			dist := zoneDistanceWithinSpace(u.zone, zone)
			bucket := far
			if u.inAttackRange(zone) {
				bucket = near
			}
			for _, target := range zone.units {
				if target.deleted || target.player_id == u.player_id {
					continue
				}
				bucket.considerUnit(target, dist)
			}
			if target := zoneEnemyBuilding(zone, u.player_id); target != nil {
				bucket.considerBuilding(target, dist)
			}
		}
	}
	if target := near.pick(u.target_priority); target != nil {
		return target
	}
	return far.pick(u.target_priority)
}

func nearbyAttackTarget(own_zone *RisqZone, player_id int, target_priority []defs.TargetCategory, in_range func(*RisqZone) bool, risq *GameRisq, space_radius uint) Attackable {
	near, far := newCategoryBest(), newCategoryBest()
	own_space := own_zone.space
	for _, space := range risq.allSpaces() {
		space_dist := game_utils.AxialDistance(own_space.coordinate, space.coordinate)
		if space_dist > space_radius || space.getVisibility(player_id) < defs.VisibilityGood {
			continue
		}
		for _, zone_row := range space.zones {
			for _, zone := range zone_row {
				dist := int(space_dist) * 6
				if space == own_space {
					dist = zoneDistanceWithinSpace(own_zone, zone)
				}
				bucket := far
				if in_range(zone) {
					bucket = near
				}
				for _, target := range zone.units {
					if target.deleted || target.player_id == player_id {
						continue
					}
					bucket.considerUnit(target, dist)
				}
				if target := zoneEnemyBuilding(zone, player_id); target != nil {
					bucket.considerBuilding(target, dist)
				}
			}
		}
	}
	if target := near.pick(target_priority); target != nil {
		return target
	}
	return far.pick(target_priority)
}

// Like nearbyAttackTarget, but never falls back to something out of range -- for stances that hold
// position instead of chasing.
func nearbyInRangeTarget(own_zone *RisqZone, player_id int, target_priority []defs.TargetCategory, in_range func(*RisqZone) bool, risq *GameRisq, space_radius uint) Attackable {
	near := newCategoryBest()
	own_space := own_zone.space
	for _, space := range risq.allSpaces() {
		space_dist := game_utils.AxialDistance(own_space.coordinate, space.coordinate)
		if space_dist > space_radius || space.getVisibility(player_id) < defs.VisibilityGood {
			continue
		}
		for _, zone_row := range space.zones {
			for _, zone := range zone_row {
				if !in_range(zone) {
					continue
				}
				dist := int(space_dist) * 6
				if space == own_space {
					dist = zoneDistanceWithinSpace(own_zone, zone)
				}
				for _, target := range zone.units {
					if target.deleted || target.player_id == player_id {
						continue
					}
					near.considerUnit(target, dist)
				}
				if target := zoneEnemyBuilding(zone, player_id); target != nil {
					near.considerBuilding(target, dist)
				}
			}
		}
	}
	return near.pick(target_priority)
}

// Returns the lowest-internal_id enemy unit in the zone, or nil
func zoneEnemyUnit(zone *RisqZone, player_id int) *RisqUnit {
	var best *RisqUnit
	for _, u := range zone.units {
		if u.deleted || u.player_id == player_id {
			continue
		}
		if best == nil || u.internal_id < best.internal_id {
			best = u
		}
	}
	return best
}

func zoneEnemyBuilding(zone *RisqZone, player_id int) *RisqBuilding {
	if zone.building == nil || zone.building.deleted || zone.building.player_id == player_id {
		return nil
	}
	return zone.building
}

func zoneHasEnemy(zone *RisqZone, player_id int) bool {
	return zoneEnemyUnit(zone, player_id) != nil || zoneEnemyBuilding(zone, player_id) != nil
}

func spaceHasEnemy(space *RisqSpace, player_id int) bool {
	for _, row := range space.zones {
		for _, zone := range row {
			if zoneHasEnemy(zone, player_id) {
				return true
			}
		}
	}
	return false
}

// Returns the zone-hop distance between two zones of the same space via BFS
func zoneDistanceWithinSpace(from *RisqZone, to *RisqZone) int {
	if from == to {
		return 0
	}
	space := from.space
	visited := map[*RisqZone]bool{from: true}
	frontier := []*RisqZone{from}
	for dist := 1; len(frontier) > 0; dist++ {
		next := []*RisqZone{}
		for _, z := range frontier {
			for _, adj := range z.adjacent_zones {
				if adj.space != space || visited[adj] {
					continue
				}
				if adj == to {
					return dist
				}
				visited[adj] = true
				next = append(next, adj)
			}
		}
		frontier = next
	}
	return -1
}

// []TargetCategory is structurally []uint8, which encoding/json marshals as a base64 string; convert to []int first.
func targetCategoriesToInts(categories []defs.TargetCategory) []int {
	ints := make([]int, len(categories))
	for i, category := range categories {
		ints[i] = int(category)
	}
	return ints
}

func targetCategoryOf(u *RisqUnit) defs.TargetCategory {
	if u.unitType() == defs.UnitType_ECONOMIC {
		return defs.TargetCategory_ECONOMIC
	}
	return defs.TargetCategory_MILITARY
}
