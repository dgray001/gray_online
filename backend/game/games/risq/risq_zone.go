package risq

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

type RisqZone struct {
	coordinate     game_utils.Coordinate2D
	coordinate_key uint
	building       *RisqBuilding
	resource       *RisqResource
	units          map[uint64]*RisqUnit
	space          *RisqSpace
	adjacent_space *RisqSpace
	adjacent_zones []*RisqZone
	ownership      int
	// index into game_utils.AxialDirectionVectors() this outer zone faces, or -1 for the center zone
	direction int
}

// Returns the direction index this zone coordinate faces, or -1 for the center zone (0, 0)
func zoneDirection(i int, j int) int {
	if i == 0 && j == 0 {
		return -1
	}
	for idx, d := range game_utils.AxialDirectionVectors() {
		if d.X == i && d.Y == j {
			return idx
		}
	}
	return -1
}

func createRisqZone(i int, j int, space *RisqSpace) *RisqZone {
	zone := RisqZone{
		coordinate:     game_utils.Coordinate2D{X: i, Y: j},
		coordinate_key: util.Pair(int(space.coordinate_key), int(util.Pair(i, j))),
		building:       nil,
		resource:       nil,
		units:          make(map[uint64]*RisqUnit, 0),
		space:          space,
		adjacent_space: nil,
		adjacent_zones: make([]*RisqZone, 0, 6),
		ownership:      -1,
		direction:      zoneDirection(i, j),
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

func (z *RisqZone) toFrontend(player_id int, v VisibilityLevel, space *RisqSpace) gin.H {
	zone := gin.H{
		"coordinate":     z.coordinate.ToFrontend(),
		"coordinate_key": z.coordinate_key,
		"ownership":      z.ownership,
	}
	if v == VisibilityFog {
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
	if z.building != nil && !z.building.deleted {
		zone["building"] = z.building.toFrontend(player_id)
	}
	if v >= VisibilityGood {
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
