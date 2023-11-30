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

func AxialDirectionVectors() [6]Coordinate2D {
	return [6]Coordinate2D{
		{X: 1, Y: 0},
		{X: 1, Y: -1},
		{X: 0, Y: -1},
		{X: -1, Y: 0},
		{X: -1, Y: 1},
		{X: 0, Y: 1},
	}
}
