package risq

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/gin-gonic/gin"
)

type RisqZone struct {
	coordinate game_utils.Coordinate2D
	building   *RisqBuilding
	resource   *RisqResource
	units      map[uint64]*RisqUnit
	space      *RisqSpace
}

func createRisqZone(i int, j int, space *RisqSpace) *RisqZone {
	zone := RisqZone{
		coordinate: game_utils.Coordinate2D{X: i, Y: j},
		building:   nil,
		resource:   nil,
		units:      make(map[uint64]*RisqUnit, 0),
		space:      space,
	}
	return &zone
}

func (z *RisqZone) toFrontend() gin.H {
	zone := gin.H{
		"coordinate": z.coordinate.ToFrontend(),
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
