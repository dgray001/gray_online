package risq

import "sort"

const buildingTickStaminaCost = 5

type ProductionIntent struct {
	order_internal_id uint64
	item              *RisqBuildingProductionItem
}

func (*ProductionIntent) isIntentKind() {}

func (i *RisqIntent) setProduction(order_internal_id uint64, item *RisqBuildingProductionItem) {
	i.detail = &ProductionIntent{order_internal_id: order_internal_id, item: item}
	i.min_cost = 1
	i.max_cost = buildingTickStaminaCost
}

// Deterministically settle same-tick unit-production completions racing for the last population slots.
func computePopulationSlotWinners(risq *GameRisq, orderables []Orderable) map[*RisqBuilding]bool {
	by_player := make(map[int][]*RisqBuilding)
	for _, o := range orderables {
		b, ok := o.(*RisqBuilding)
		if !ok || !b.intent.hasIntent() {
			continue
		}
		production, ok := b.intent.detail.(*ProductionIntent)
		if !ok || production.item.kind != ProducibleKind_UNIT || production.item.stamina_remaining-b.intent.intent_cost > 0 {
			continue
		}
		by_player[b.player_id] = append(by_player[b.player_id], b)
	}
	winners := make(map[*RisqBuilding]bool)
	for player_id, buildings := range by_player {
		sort.Slice(buildings, func(i, j int) bool { return buildings[i].internal_id < buildings[j].internal_id })
		player := risq.players[player_id]
		remaining := int(player.populationLimit()) - nonDeletedUnitCount(player.units)
		for i, b := range buildings {
			if i >= remaining {
				break
			}
			winners[b] = true
		}
	}
	return winners
}

// One garrisoned unit's own attack riding along with the building's attack this tick
type GarrisonAttack struct {
	unit  *RisqUnit
	stats RisqCombatStats
	cost  int
}

// Wraps a garrisoned unit so it attacks with its precomputed, capped stats instead of recomputing them
type garrisonAttacker struct {
	*RisqUnit
	stats RisqCombatStats
}

func (g garrisonAttacker) combatStats(r *GameRisq, other Orderable, attacking bool) RisqCombatStats {
	return g.stats
}

type BuildingAttackIntent struct {
	target           Attackable
	garrison_attacks []GarrisonAttack
}

func (*BuildingAttackIntent) isIntentKind() {}

func (i *RisqIntent) setBuildingAttack(target Attackable, garrison_attacks []GarrisonAttack) {
	i.detail = &BuildingAttackIntent{target: target, garrison_attacks: garrison_attacks}
	if len(garrison_attacks) > 0 {
		i.min_cost = 0
	} else {
		i.min_cost = 1
	}
	i.max_cost = unitTickStaminaCost
}
