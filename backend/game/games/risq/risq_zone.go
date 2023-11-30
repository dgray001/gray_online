package risq

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/gin-gonic/gin"
)

type RisqZone struct {
	coordinate game_utils.Coordinate2D
	building   *RisqBuilding
	units      []*RisqUnit
}

func createRisqZone(i int, j int) *RisqZone {
	zone := RisqZone{
		coordinate: game_utils.Coordinate2D{X: i, Y: j},
		building:   nil,
		units:      make([]*RisqUnit, 0),
	}
	return &zone
}

func (s *RisqZone) toFrontend() gin.H {
	zone := gin.H{
		"coordinate": s.coordinate.ToFrontend(),
	}
	if s.building != nil && !s.building.deleted {
		zone["building"] = s.building.toFrontend()
	}
	units := []gin.H{}
	for _, unit := range s.units {
		if unit != nil {
			units = append(units, unit.toFrontend())
		}
	}
	zone["units"] = units
	return zone
}
