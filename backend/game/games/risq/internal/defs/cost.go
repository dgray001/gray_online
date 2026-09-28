package defs

import (
	"github.com/gin-gonic/gin"
)

type RisqResourceCost struct {
	Food  float64
	Wood  float64
	Stone float64
	Gold  float64
}

func (c RisqResourceCost) ToFrontend() gin.H {
	return gin.H{
		"food":  c.Food,
		"wood":  c.Wood,
		"stone": c.Stone,
		"gold":  c.Gold,
	}
}

func (c RisqResourceCost) Scale(f float64) RisqResourceCost {
	return RisqResourceCost{
		Food:  c.Food * f,
		Wood:  c.Wood * f,
		Stone: c.Stone * f,
		Gold:  c.Gold * f,
	}
}

func (c RisqResourceCost) Points() uint {
	return uint(c.Food + c.Wood + c.Stone + 2*c.Gold)
}
