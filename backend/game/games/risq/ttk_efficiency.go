package risq

import (
	"math"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

const (
	killRateSamples  = 200
	maxKillRateTurns = 100000
)

func UnitCost(id uint32) float64 {
	cost := defs.DefaultConfig().Units[id].Cost
	return cost.Food + cost.Wood + cost.Stone + cost.Gold
}

// Lanchester square-law multiple: the budget of b that a's budget beats; above 1 means a counters b at equal cost
func ComputeCostEfficiencyMultiple(a uint32, b uint32, a_tech_ids []uint32, b_tech_ids []uint32) (float64, bool) {
	a_rate := ComputeUnitKillRate(a, b, a_tech_ids, b_tech_ids)
	b_rate := ComputeUnitKillRate(b, a, b_tech_ids, a_tech_ids)
	if a_rate == 0 || b_rate == 0 {
		return 0, false
	}
	return (math.Sqrt(a_rate) / UnitCost(a)) / (math.Sqrt(b_rate) / UnitCost(b)), true
}

// Steady-state kills per turn of one unit chaining kills, with unspent stamina carrying into the next kill
func ComputeUnitKillRate(attacker_unit_id uint32, target_unit_id uint32, attacker_tech_ids []uint32, target_tech_ids []uint32) float64 {
	config := defs.DefaultConfig()
	_, attacker_known := config.Units[attacker_unit_id]
	_, target_known := config.Units[target_unit_id]
	if !attacker_known || !target_known {
		return 0
	}
	r, attacker_player, target_player := newTTKGame(config)
	markResearched(attacker_player, attacker_tech_ids)
	markResearched(target_player, target_tech_ids)
	attacker := createRisqUnit(1, attacker_unit_id, attacker_player)
	target := createRisqUnit(2, target_unit_id, target_player)
	space := r.spaces[0][0]
	space.setUnit(&game_utils.Coordinate2D{}, attacker)
	space.setUnit(&game_utils.Coordinate2D{}, target)
	kills, spent := 0, 0
	for turn := 0; turn < maxKillRateTurns && kills < killRateSamples; turn++ {
		attacker.refreshStamina()
		for attacker.current_stamina > 0 && kills < killRateSamples {
			cost := min(attacker.current_stamina, unitTickStaminaCost)
			before := target.cs.health
			attacker.intent.intent_cost = cost
			r.unitAttack(attacker, target)
			target.resolveHealthDelta(r)
			if target.cs.health == before {
				return 0
			}
			attacker.current_stamina -= cost
			spent += cost
			if !target.isAlive() {
				kills++
				target.cs.health = float64(target.cs.max_health)
			}
		}
	}
	return float64(attacker.turn_stamina*kills) / float64(spent)
}
