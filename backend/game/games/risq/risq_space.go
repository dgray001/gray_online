package risq

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/gin-gonic/gin"
)

type RisqSpace struct {
	coordinate game_utils.Coordinate2D
}

func createRisqSpace(i int, j int) *RisqSpace {
	return &RisqSpace{
		coordinate: game_utils.Coordinate2D{X: i, Y: j},
	}
}

func (s *RisqSpace) toFrontend() gin.H {
	space := gin.H{
		"coordinate": s.coordinate.ToFrontend(),
	}
	return space
}
