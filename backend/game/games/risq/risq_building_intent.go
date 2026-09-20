package risq

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
