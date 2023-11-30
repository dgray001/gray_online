package risq

import "github.com/gin-gonic/gin"

type RisqBuilding struct {
	deleted bool
}

func (b *RisqBuilding) toFrontend() gin.H {
	building := gin.H{}
	return building
}
