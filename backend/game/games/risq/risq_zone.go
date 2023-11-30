package risq

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/gin-gonic/gin"
)

type RisqZone struct {
	coordinate game_utils.Coordinate2D
	building   *RisqBuilding
	units      map[uint64]*RisqUnit
}

func createRisqZone(i int, j int) *RisqZone {
	zone := RisqZone{
		coordinate: game_utils.Coordinate2D{X: i, Y: j},
		building:   nil,
		units:      make(map[uint64]*RisqUnit, 0),
	}
	return &zone
}

func (z *RisqZone) toFrontend() gin.H {
	zone := gin.H{
		"coordinate": z.coordinate.ToFrontend(),
	}
	if z.building != nil && !z.building.deleted {
		zone["building"] = z.building.toFrontend()
	}
	units := make(map[uint64]gin.H, 0)
	for _, unit := range z.units {
		if unit != nil && !unit.deleted {
			units[unit.internal_id] = unit.toFrontend()
		}
	}
	zone["units"] = units
	return zone
}
