package game_utils

import "github.com/gin-gonic/gin"

type Coordinate2D struct {
	X int
	Y int
}

func (c *Coordinate2D) ToFrontend() gin.H {
	coordinate := gin.H{
		"x": c.X,
		"y": c.Y,
	}
	return coordinate
}
