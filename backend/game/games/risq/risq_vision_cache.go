package risq

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/gin-gonic/gin"
)

// frozen snapshot from the last turn with >= poor vision; omits order/production data (spy-tier only)
type RisqBuildingCache struct {
	internal_id                uint64
	player_id                  int
	building_id                uint32
	display_name               string
	population_support         uint16
	cs                         RisqCombatStats
	under_construction         bool
	stamina_remaining          int
	construction_stamina_total int
	zone_coordinate            game_utils.Coordinate2D
	space_coordinate           game_utils.Coordinate2D
}

func cacheRisqBuilding(b *RisqBuilding) RisqBuildingCache {
	cache := RisqBuildingCache{
		internal_id:                b.internal_id,
		player_id:                  b.player_id,
		building_id:                b.building_id,
		display_name:               b.display_name,
		population_support:         b.population_support,
		cs:                         b.cs,
		under_construction:         b.underConstruction(),
		stamina_remaining:          b.stamina_remaining,
		construction_stamina_total: b.construction_stamina_total,
	}
	if b.zone != nil {
		cache.zone_coordinate = b.zone.coordinate
		if b.zone.space != nil {
			cache.space_coordinate = b.zone.space.coordinate
		}
	}
	return cache
}

func (c RisqBuildingCache) toFrontend() gin.H {
	return gin.H{
		"internal_id":                c.internal_id,
		"player_id":                  c.player_id,
		"building_id":                c.building_id,
		"display_name":               c.display_name,
		"population_support":         c.population_support,
		"combat_stats":               c.cs.toFrontend(),
		"under_construction":         c.under_construction,
		"stamina_remaining":          c.stamina_remaining,
		"construction_stamina_total": c.construction_stamina_total,
		"turn_stamina":               0,
		"current_stamina":            0,
		"max_stamina":                0,
		"produces":                   buildingProducesToFrontend(c.building_id),
		"active_orders":              []gin.H{},
		"production_queue":           []gin.H{},
		"zone_coordinate":            c.zone_coordinate.ToFrontend(),
		"space_coordinate":           c.space_coordinate.ToFrontend(),
	}
}

// frozen snapshot from the last turn with >= poor vision
type RisqResourceCache struct {
	internal_id       uint64
	resource_id       uint32
	display_name      string
	resources_left    float64
	base_gather_speed int
	zone_coordinate   game_utils.Coordinate2D
	space_coordinate  game_utils.Coordinate2D
}

func cacheRisqResource(r *RisqResource) RisqResourceCache {
	cache := RisqResourceCache{
		internal_id:       r.internal_id,
		resource_id:       r.resource_id,
		display_name:      r.display_name,
		resources_left:    r.resources_left,
		base_gather_speed: r.base_gather_speed,
	}
	if r.zone != nil {
		cache.zone_coordinate = r.zone.coordinate
		if r.zone.space != nil {
			cache.space_coordinate = r.zone.space.coordinate
		}
	}
	return cache
}

func (c RisqResourceCache) toFrontend() gin.H {
	return gin.H{
		"internal_id":       c.internal_id,
		"resource_id":       c.resource_id,
		"display_name":      c.display_name,
		"resources_left":    c.resources_left,
		"base_gather_speed": c.base_gather_speed,
		"zone_coordinate":   c.zone_coordinate.ToFrontend(),
		"space_coordinate":  c.space_coordinate.ToFrontend(),
	}
}
