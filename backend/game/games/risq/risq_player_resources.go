package risq

import "github.com/gin-gonic/gin"

type RisqPlayerResources struct {
	food  float64
	wood  float64
	stone float64
	gold  float64
}

type RisqResourceCost struct {
	food  float64
	wood  float64
	stone float64
	gold  float64
}

func (c RisqResourceCost) toFrontend() gin.H {
	return gin.H{
		"food":  c.food,
		"wood":  c.wood,
		"stone": c.stone,
		"gold":  c.gold,
	}
}

func (c RisqResourceCost) times(n int) RisqResourceCost {
	return RisqResourceCost{
		food:  c.food * float64(n),
		wood:  c.wood * float64(n),
		stone: c.stone * float64(n),
		gold:  c.gold * float64(n),
	}
}

func (c RisqResourceCost) scale(f float64) RisqResourceCost {
	return RisqResourceCost{
		food:  c.food * f,
		wood:  c.wood * f,
		stone: c.stone * f,
		gold:  c.gold * f,
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
	case RisqResourceCategory_GOLD:
		r.gold += amount
	}
}

func (r *RisqPlayerResources) canAfford(cost RisqResourceCost) bool {
	return r.food >= cost.food && r.wood >= cost.wood && r.stone >= cost.stone && r.gold >= cost.gold
}

// Returns the fraction of cost affordable in [0, 1] (1 if cost is free)
func (r *RisqPlayerResources) affordFraction(cost RisqResourceCost) float64 {
	fraction := 1.0
	limit := func(available float64, needed float64) {
		if needed > 0 && available/needed < fraction {
			fraction = available / needed
		}
	}
	limit(r.food, cost.food)
	limit(r.wood, cost.wood)
	limit(r.stone, cost.stone)
	limit(r.gold, cost.gold)
	if fraction < 0 {
		return 0
	}
	return fraction
}

func (r *RisqPlayerResources) spend(cost RisqResourceCost) {
	r.food -= cost.food
	r.wood -= cost.wood
	r.stone -= cost.stone
	r.gold -= cost.gold
}

func (r *RisqPlayerResources) refund(cost RisqResourceCost) {
	r.food += cost.food
	r.wood += cost.wood
	r.stone += cost.stone
	r.gold += cost.gold
}

func (r *RisqPlayerResources) score() uint {
	return uint(r.food + r.wood + r.stone + r.gold)
}

func (r *RisqPlayerResources) toFrontend() gin.H {
	resources := gin.H{
		"food":  r.food,
		"wood":  r.wood,
		"stone": r.stone,
		"gold":  r.gold,
	}
	return resources
}
