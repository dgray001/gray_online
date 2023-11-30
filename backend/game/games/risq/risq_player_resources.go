package risq

import "github.com/gin-gonic/gin"

type RisqPlayerResources struct {
	food uint16
	wood uint16
}

func createRisqPlayerResources() *RisqPlayerResources {
	return &RisqPlayerResources{
		food: 0,
		wood: 0,
	}
}

func (r *RisqPlayerResources) toFrontend() gin.H {
	resources := gin.H{
		"food": r.food,
		"wood": r.wood,
	}
	return resources
}
