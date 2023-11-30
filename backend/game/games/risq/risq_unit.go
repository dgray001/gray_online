package risq

import "github.com/gin-gonic/gin"

type RisqUnit struct {
	deleted bool
}

func (u *RisqUnit) toFrontend() gin.H {
	unit := gin.H{}
	return unit
}
