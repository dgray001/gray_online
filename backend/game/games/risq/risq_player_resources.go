package risq

import "github.com/gin-gonic/gin"

type RisqPlayerResources struct {
	food  uint16
	wood  uint16
	stone uint16
}

func createRisqPlayerResources() *RisqPlayerResources {
	return &RisqPlayerResources{
		food:  0,
		wood:  0,
		stone: 0,
	}
}

func (r *RisqPlayerResources) score() uint {
	return uint(r.food) + uint(r.wood) + uint(r.stone)
}

func (r *RisqPlayerResources) toFrontend() gin.H {
	resources := gin.H{
		"food":  r.food,
		"wood":  r.wood,
		"stone": r.stone,
	}
	return resources
}
