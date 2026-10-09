package risq

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

type RisqZone struct {
	coordinate               game_utils.Coordinate2D
	coordinate_key           uint
	building                 *RisqBuilding
	destroyed_building       uint32
	destroyed_building_turns uint8
	resource                 *RisqResource
	units                    map[uint64]*RisqUnit
	corpses                  map[uint64]RisqCorpse
	space                    *RisqSpace
	adjacent_space           *RisqSpace
	adjacent_zones           []*RisqZone
	ownership                int
	// index into game_utils.AxialDirectionVectors() this outer zone faces, or -1 for the center zone
	direction int
	// cosmetic-only terrain_id override; 0 means none
	terrain_override uint32
}

// Returns the direction index this zone coordinate faces, or -1 for the center zone (0, 0)
func createRisqZone(i int, j int, space *RisqSpace) *RisqZone {
	zone := RisqZone{
		coordinate:     game_utils.Coordinate2D{X: i, Y: j},
		coordinate_key: util.Pair(int(space.coordinate_key), int(util.Pair(i, j))),
		building:       nil,
		resource:       nil,
		units:          make(map[uint64]*RisqUnit, 0),
		corpses:        make(map[uint64]RisqCorpse),
		space:          space,
		adjacent_space: nil,
		adjacent_zones: make([]*RisqZone, 0, 6),
		ownership:      -1,
		direction:      game_utils.AxialDirectionIndex(game_utils.Coordinate2D{X: i, Y: j}),
	}
	return &zone
}

func invertZoneKey(k uint, r *GameRisq) (*RisqSpace, *RisqZone) {
	space_key, zone_key := util.InvertPair(k)
	x, y := util.InvertPair(uint(space_key))
	space := r.getSpace(&game_utils.Coordinate2D{X: x, Y: y})
	if space == nil {
		return nil, nil
	}
	i, j := util.InvertPair(uint(zone_key))
	zone := space.getZone(&game_utils.Coordinate2D{X: i, Y: j})
	return space, zone
}

func invertBuildKey(k uint, r *GameRisq) (uint32, *RisqSpace, *RisqZone) {
	building_id, zone_key := util.InvertPair(k)
	space, zone := invertZoneKey(uint(zone_key), r)
	return uint32(building_id), space, zone
}

func (z *RisqZone) isCenter() bool {
	return z.direction < 0
}

func (z *RisqZone) updateRubble(r *GameRisq, advance bool) {
	if z.destroyed_building == 0 {
		return
	}
	if advance {
		z.destroyed_building_turns++
	}
	occupied := z.building != nil || z.resource != nil
	for _, player := range r.players {
		occupied = occupied || player.planned_foundations[z.coordinate_key] != nil
	}
	if occupied || z.destroyed_building_turns > 2 {
		z.destroyed_building = 0
		z.destroyed_building_turns = 0
	}
}

func (z *RisqZone) toFrontend(player_id int, v defs.VisibilityLevel, space *RisqSpace) gin.H {
	terrainOverride := uint32(0)
	if v == defs.VisibilityFog {
		terrainOverride = space.terrain_cache[player_id][z.coordinate_key]
	} else if v != defs.VisibilityUnexplored {
		terrainOverride = z.terrain_override
	}
	zone := gin.H{
		"coordinate":       z.coordinate.ToFrontend(),
		"coordinate_key":   z.coordinate_key,
		"terrain_override": terrainOverride,
	}
	if v != defs.VisibilityFog {
		zone["ownership"] = z.ownership
	} else if owner, ok := space.ownership_cache[player_id]; ok {
		zone["ownership"] = owner
	}
	if terrainOverride != 0 {
		zone["terrain_override_display_name"] = defs.TerrainConfigs[terrainOverride].Display_name
	}
	if v == defs.VisibilityFog {
		if cache, ok := space.rubble_cache[player_id][z.coordinate_key]; ok {
			zone["destroyed_building"] = cache.building_id
			zone["destroyed_building_turns"] = cache.turns
		}
		if cache, ok := space.resource_cache[player_id][z.coordinate_key]; ok {
			zone["resource"] = cache.toFrontend()
		}
		if cache, ok := space.building_cache[player_id][z.coordinate_key]; ok {
			zone["building"] = cache.toFrontend()
		}
		return zone
	}
	if z.resource != nil && z.resource.resources_left > 0 {
		zone["resource"] = z.resource.toFrontend()
	}
	if v >= defs.VisibilityPoor && z.destroyed_building != 0 {
		zone["destroyed_building"] = z.destroyed_building
		zone["destroyed_building_turns"] = z.destroyed_building_turns
	}
	if z.building != nil && !z.building.deleted {
		zone["building"] = z.building.toFrontend(player_id)
	}
	if v >= defs.VisibilityGood {
		if len(z.corpses) > 0 {
			zone["corpses"] = z.corpsesToFrontend()
		}
		units := make([]gin.H, 0)
		for _, unit := range z.units {
			if unit != nil && !unit.deleted {
				units = append(units, unit.toFrontend(player_id))
			}
		}
		zone["units"] = units
	} else {
		zone["unit_count"] = nonDeletedUnitCount(z.units)
	}
	return zone
}
