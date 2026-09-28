package risq

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

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

func createRisqPlayerResources() *RisqPlayerResources {
	return &RisqPlayerResources{food: 200, wood: 200, stone: 50, gold: 2000}
}

func (r *RisqPlayerResources) addGathered(category defs.RisqResourceCategory, amount float64) {
	switch category {
	case defs.RisqResourceCategory_FOOD:
		r.food = util.RoundTo(r.food+amount, gatherRoundingPlaces)
	case defs.RisqResourceCategory_WOOD:
		r.wood = util.RoundTo(r.wood+amount, gatherRoundingPlaces)
	case defs.RisqResourceCategory_STONE:
		r.stone = util.RoundTo(r.stone+amount, gatherRoundingPlaces)
	case defs.RisqResourceCategory_GOLD:
		r.gold = util.RoundTo(r.gold+amount, gatherRoundingPlaces)
	}
	r.gathered[category.Index()] = util.RoundTo(r.gathered[category.Index()]+amount, gatherRoundingPlaces)
	r.lifetime_gathered[category.Index()] = util.RoundTo(r.lifetime_gathered[category.Index()]+amount, gatherRoundingPlaces)
}

func (r *RisqPlayerResources) LifetimeGathered() defs.RisqResourceCost {
	return defs.RisqResourceCost{
		Food:  r.lifetime_gathered[defs.RisqResourceCategory_FOOD.Index()],
		Wood:  r.lifetime_gathered[defs.RisqResourceCategory_WOOD.Index()],
		Stone: r.lifetime_gathered[defs.RisqResourceCategory_STONE.Index()],
		Gold:  r.lifetime_gathered[defs.RisqResourceCategory_GOLD.Index()],
	}
}

func (r *RisqPlayerResources) canAfford(cost defs.RisqResourceCost) bool {
	return r.food >= cost.Food && r.wood >= cost.Wood && r.stone >= cost.Stone && r.gold >= cost.Gold
}

// Returns the fraction of cost affordable in [0, 1] (1 if cost is free)
func (r *RisqPlayerResources) affordFraction(cost defs.RisqResourceCost) float64 {
	fraction := 1.0
	limit := func(available float64, needed float64) {
		if needed > 0 && available/needed < fraction {
			fraction = available / needed
		}
	}
	limit(r.food, cost.Food)
	limit(r.wood, cost.Wood)
	limit(r.stone, cost.Stone)
	limit(r.gold, cost.Gold)
	if fraction < 0 {
		return 0
	}
	return fraction
}

func (r *RisqPlayerResources) spend(cost defs.RisqResourceCost) {
	r.food -= cost.Food
	r.wood -= cost.Wood
	r.stone -= cost.Stone
	r.gold -= cost.Gold
	r.spent[defs.RisqResourceCategory_FOOD.Index()] += cost.Food
	r.spent[defs.RisqResourceCategory_WOOD.Index()] += cost.Wood
	r.spent[defs.RisqResourceCategory_STONE.Index()] += cost.Stone
	r.spent[defs.RisqResourceCategory_GOLD.Index()] += cost.Gold
}

func (r *RisqPlayerResources) refund(cost defs.RisqResourceCost) {
	r.food += cost.Food
	r.wood += cost.Wood
	r.stone += cost.Stone
	r.gold += cost.Gold
	r.gathered[defs.RisqResourceCategory_FOOD.Index()] += cost.Food
	r.gathered[defs.RisqResourceCategory_WOOD.Index()] += cost.Wood
	r.gathered[defs.RisqResourceCategory_STONE.Index()] += cost.Stone
	r.gathered[defs.RisqResourceCategory_GOLD.Index()] += cost.Gold
}

func (r *RisqPlayerResources) resetFlow() {
	r.gathered = [4]float64{}
	r.spent = [4]float64{}
}

func (r *RisqPlayerResources) score() uint {
	return uint(r.food + r.wood + r.stone + 2*r.gold)
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
