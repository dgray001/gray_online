package risq

import (
	"cmp"
	"maps"
	"math"
	"slices"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
)

func (b *RisqBuilding) repairHealthValue(health_units int64) defs.RisqResourceCost {
	cost := defs.BuildingConfigs[b.building_id].Cost.Scale(repairCostFactor * float64(health_units) / (10000 * float64(b.cs.max_health)))
	return defs.RisqResourceCost{
		Food: util.RoundTo(cost.Food, gatherRoundingPlaces), Wood: util.RoundTo(cost.Wood, gatherRoundingPlaces),
		Stone: util.RoundTo(cost.Stone, gatherRoundingPlaces), Gold: util.RoundTo(cost.Gold, gatherRoundingPlaces),
	}
}

func (b *RisqBuilding) repairCharge(health_units int64) defs.RisqResourceCost {
	starting_health := int64(math.Round(b.cs.health * 10000))
	before := b.repairHealthValue(starting_health)
	after := b.repairHealthValue(starting_health + health_units)
	return defs.RisqResourceCost{
		Food: util.RoundTo(after.Food-before.Food, gatherRoundingPlaces), Wood: util.RoundTo(after.Wood-before.Wood, gatherRoundingPlaces),
		Stone: util.RoundTo(after.Stone-before.Stone, gatherRoundingPlaces), Gold: util.RoundTo(after.Gold-before.Gold, gatherRoundingPlaces),
	}
}

func (b *RisqBuilding) affordableRepairHealth(resources *RisqPlayerResources, health_units int64) int64 {
	low, high := int64(0), health_units
	for low < high {
		mid := low + (high-low+1)/2
		if resources.canAfford(b.repairCharge(mid)) {
			low = mid
		} else {
			high = mid - 1
		}
	}
	return low
}

func (r *GameRisq) resolveRepairs(orderables []Orderable) {
	healing := make(map[*RisqBuilding]float64)
	for _, o := range slices.SortedFunc(slices.Values(orderables), func(a, b Orderable) int { return cmp.Compare(a.internalId(), b.internalId()) }) {
		u, ok := o.(*RisqUnit)
		if !ok || !u.intent.hasIntent() {
			continue
		}
		repair, ok := u.intent.detail.(*RepairIntent)
		if !ok || repair.target.deleted || repair.target.underConstruction() || !r.canAssist(u.player_id, repair.target) {
			continue
		}
		if heal, _, ok := repairHealAndCost(repair.target, u.intent.intent_cost); ok {
			healing[repair.target] += heal * r.repair_allotments[u]
		}
	}
	for _, b := range slices.SortedFunc(maps.Keys(healing), func(a, b *RisqBuilding) int { return cmp.Compare(a.internal_id, b.internal_id) }) {
		health_units := int64(math.Round(min(healing[b], float64(b.cs.max_health)-b.cs.health) * 10000))
		resources := r.players[b.player_id].resources
		health_units = b.affordableRepairHealth(resources, health_units)
		resources.spend(b.repairCharge(health_units))
		b.cs.queueHealth(float64(health_units) / 10000)
	}
}
