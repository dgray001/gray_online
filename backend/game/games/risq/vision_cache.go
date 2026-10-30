package risq

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
)

type RisqRubbleCache struct {
	building_id uint32
	turns       uint8
}

// frozen snapshot from the last turn with >= poor vision; omits order/production data (spy-tier only)
type RisqBuildingCache struct {
	config                     *defs.Config
	internal_id                uint64
	player_id                  int
	building_id                uint32
	display_name               string
	population_support         uint16
	cs                         RisqCombatStats
	under_construction         bool
	has_garrisoned_units       bool
	stamina_remaining          int
	construction_stamina_total int
	zone_coordinate            game_utils.Coordinate2D
	space_coordinate           game_utils.Coordinate2D
}

func cacheRisqBuilding(b *RisqBuilding) RisqBuildingCache {
	cache := RisqBuildingCache{
		config:                     b.config,
		internal_id:                b.internal_id,
		player_id:                  b.player_id,
		building_id:                b.building_id,
		display_name:               b.display_name,
		population_support:         b.population_support,
		cs:                         b.cs,
		under_construction:         b.underConstruction(),
		has_garrisoned_units:       len(b.garrisoned_units) > 0,
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
		"produces":                   buildingProducesToFrontend(c.config, c.building_id),
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
	max_resources     float64
	base_gather_speed int
	gather_capacity   int
	zone_coordinate   game_utils.Coordinate2D
	space_coordinate  game_utils.Coordinate2D
}

func cacheRisqResource(r *RisqResource) RisqResourceCache {
	cache := RisqResourceCache{
		internal_id:       r.internal_id,
		resource_id:       r.resource_id,
		display_name:      r.display_name,
		resources_left:    r.resources_left,
		max_resources:     r.max_resources,
		base_gather_speed: r.base_gather_speed,
		gather_capacity:   r.gather_capacity,
	}
	if r.zone != nil {
		cache.zone_coordinate = r.zone.coordinate
		if r.zone.space != nil {
			cache.space_coordinate = r.zone.space.coordinate
		}
	}
	return cache
}

// Resource info about a zone the player currently or previously has vision of
func (z *RisqZone) resourceKnownTo(player_id int) (RisqResourceCache, bool) {
	switch z.space.getVisibility(player_id) {
	case defs.VisibilityFog:
		cache, ok := z.space.resource_cache[player_id][z.coordinate_key]
		return cache, ok
	case defs.VisibilityUnexplored:
		return RisqResourceCache{}, false
	default:
		if z.resource == nil {
			return RisqResourceCache{}, false
		}
		return cacheRisqResource(z.resource), true
	}
}

// Building info about a zone the player currently or previously has vision of
func (z *RisqZone) buildingKnownTo(player_id int) (RisqBuildingCache, bool) {
	switch z.space.getVisibility(player_id) {
	case defs.VisibilityFog:
		cache, ok := z.space.building_cache[player_id][z.coordinate_key]
		return cache, ok
	case defs.VisibilityUnexplored:
		return RisqBuildingCache{}, false
	default:
		if z.building == nil || z.building.deleted {
			return RisqBuildingCache{}, false
		}
		return cacheRisqBuilding(z.building), true
	}
}

func (r *GameRisq) buildingExistsOrRemembered(building_id uint64, player_id int) bool {
	if r.buildings[building_id] != nil {
		return true
	}
	for _, row := range r.spaces {
		for _, space := range row {
			if space == nil || space.getVisibility(player_id) != defs.VisibilityFog {
				continue
			}
			for _, cache := range space.building_cache[player_id] {
				if cache.internal_id == building_id {
					return true
				}
			}
		}
	}
	return false
}

func (c RisqResourceCache) toFrontend() gin.H {
	return gin.H{
		"internal_id":       c.internal_id,
		"resource_id":       c.resource_id,
		"display_name":      c.display_name,
		"resources_left":    c.resources_left,
		"max_resources":     c.max_resources,
		"base_gather_speed": c.base_gather_speed,
		"gather_capacity":   c.gather_capacity,
		"zone_coordinate":   c.zone_coordinate.ToFrontend(),
		"space_coordinate":  c.space_coordinate.ToFrontend(),
	}
}

func showOrdersTo(subject_player_id int, zone *RisqZone, viewer_player_id int) bool {
	if subject_player_id == viewer_player_id {
		return true
	}
	if zone == nil || zone.space == nil {
		return false
	}
	return zone.space.getVisibility(viewer_player_id) >= defs.VisibilitySpy
}

func (r *GameRisq) initializeVisibility() {
	level := r.visibility_mode.MinimumVision()
	if level == defs.VisibilityUnexplored {
		return
	}
	for _, space := range r.allSpaces() {
		for _, player := range r.players {
			player_id := player.player.Player_id
			space.visibility[player_id] = max(space.getVisibility(player_id), level)
			space.refreshCache(player_id)
		}
	}
}
