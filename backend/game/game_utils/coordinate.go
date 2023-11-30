package game_utils

import "github.com/gin-gonic/gin"

type Coordinate2D struct {
	X int
	Y int
}

func (c1 *Coordinate2D) Add(c2 *Coordinate2D) *Coordinate2D {
	return &Coordinate2D{
		X: c1.X + c2.X,
		Y: c1.Y + c2.Y,
	}
}

func (c1 *Coordinate2D) Subtract(c2 *Coordinate2D) *Coordinate2D {
	return &Coordinate2D{
		X: c1.X - c2.X,
		Y: c1.Y - c2.Y,
	}
}

func (c *Coordinate2D) Multiply(f int) *Coordinate2D {
	return &Coordinate2D{
		X: f * c.X,
		Y: f * c.Y,
	}
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
