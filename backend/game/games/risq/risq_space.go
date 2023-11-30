package risq

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

type RisqSpace struct {
	coordinate game_utils.Coordinate2D
	zones      [][]*RisqZone
}

func createRisqSpace(i int, j int) *RisqSpace {
	space := RisqSpace{
		coordinate: game_utils.Coordinate2D{X: i, Y: j},
	}
	space.zones = make([][]*RisqZone, 3)
	for j := range space.zones {
		r := j - 1
		l := 3 - util.AbsInt(r)
		space.zones[j] = make([]*RisqZone, l)
		for i := range space.zones[j] {
			q := max(-1, -(1+r)) + i
			space.zones[j][i] = createRisqZone(q, r)
		}
	}
	return &space
}

func (s *RisqSpace) toFrontend(player_id int, is_viewer bool) gin.H {
	space := gin.H{
		"coordinate": s.coordinate.ToFrontend(),
	}
	zones := [][]gin.H{}
	for _, row := range s.zones {
		zones_row := []gin.H{}
		for _, zone := range row {
			zones_row = append(zones_row, zone.toFrontend())
		}
		zones = append(zones, zones_row)
	}
	space["zones"] = zones
	return space
}
