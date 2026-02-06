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
	}
	return &zone
}

func (z *RisqZone) isCenter() bool {
	return z.coordinate.X == 0 && z.coordinate.Y == 0
}

func (z *RisqZone) toFrontend() gin.H {
	zone := gin.H{
		"coordinate":     z.coordinate.ToFrontend(),
		"coordinate_key": z.coordinate_key,
	}
	if z.resource != nil && z.resource.resources_left > 0 {
		zone["resource"] = z.resource.toFrontend()
	}
	if z.building != nil && !z.building.deleted {
		zone["building"] = z.building.toFrontend()
	}
	units := make([]gin.H, 0)
	for _, unit := range z.units {
		if unit != nil && !unit.deleted {
			units = append(units, unit.toFrontend())
		}
	}
	zone["units"] = units
	return zone
}
