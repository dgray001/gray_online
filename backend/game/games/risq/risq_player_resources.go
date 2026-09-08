package risq

import "github.com/gin-gonic/gin"

type RisqPlayerResources struct {
	food  float64
	wood  float64
	stone float64
	gold  float64
	// per-turn flow by category, reset each turn
	gathered [4]float64
	spent    [4]float64
	// cumulative gathered (not refunds) across the whole game, never reset
	lifetime_gathered [4]float64
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
	return &RisqPlayerResources{food: 200, wood: 100, stone: 50}
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
	r.gathered[category] += amount
	r.lifetime_gathered[category] += amount
}

func (r *RisqPlayerResources) LifetimeGathered() RisqResourceCost {
	return RisqResourceCost{
		food:  r.lifetime_gathered[RisqResourceCategory_FOOD],
		wood:  r.lifetime_gathered[RisqResourceCategory_WOOD],
		stone: r.lifetime_gathered[RisqResourceCategory_STONE],
		gold:  r.lifetime_gathered[RisqResourceCategory_GOLD],
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
	r.spent[RisqResourceCategory_FOOD] += cost.food
	r.spent[RisqResourceCategory_WOOD] += cost.wood
	r.spent[RisqResourceCategory_STONE] += cost.stone
	r.spent[RisqResourceCategory_GOLD] += cost.gold
}

func (r *RisqPlayerResources) refund(cost RisqResourceCost) {
	r.food += cost.food
	r.wood += cost.wood
	r.stone += cost.stone
	r.gold += cost.gold
	r.gathered[RisqResourceCategory_FOOD] += cost.food
	r.gathered[RisqResourceCategory_WOOD] += cost.wood
	r.gathered[RisqResourceCategory_STONE] += cost.stone
	r.gathered[RisqResourceCategory_GOLD] += cost.gold
}

func (r *RisqPlayerResources) resetFlow() {
	r.gathered = [4]float64{}
	r.spent = [4]float64{}
}

func (r *RisqPlayerResources) score() uint {
	return uint(r.food + r.wood + r.stone + 2*r.gold)
}

func (c RisqResourceCost) points() uint {
	return uint(c.food + c.wood + c.stone + 2*c.gold)
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
