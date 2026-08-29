package risq

import "github.com/gin-gonic/gin"

type RisqPlayerResources struct {
	food  float64
	wood  float64
	stone float64
}

type RisqResourceCost struct {
	food  float64
	wood  float64
	stone float64
}

func (c RisqResourceCost) times(n int) RisqResourceCost {
	return RisqResourceCost{
		food:  c.food * float64(n),
		wood:  c.wood * float64(n),
		stone: c.stone * float64(n),
	}
}

func createRisqPlayerResources() *RisqPlayerResources {
	return &RisqPlayerResources{}
}

func (r *RisqPlayerResources) addGathered(category RisqResourceCategory, amount float64) {
	switch category {
	case RisqResourceCategory_FOOD:
		r.food += amount
	case RisqResourceCategory_WOOD:
		r.wood += amount
	case RisqResourceCategory_STONE:
		r.stone += amount
	}
}

func (r *RisqPlayerResources) canAfford(cost RisqResourceCost) bool {
	return r.food >= cost.food && r.wood >= cost.wood && r.stone >= cost.stone
}

func (r *RisqPlayerResources) spend(cost RisqResourceCost) {
	r.food -= cost.food
	r.wood -= cost.wood
	r.stone -= cost.stone
}

func (r *RisqPlayerResources) score() uint {
	return uint(r.food + r.wood + r.stone)
}

func (r *RisqPlayerResources) toFrontend() gin.H {
	resources := gin.H{
		"food":  r.food,
		"wood":  r.wood,
		"stone": r.stone,
	}
	return resources
}
