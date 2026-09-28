package risq

import "github.com/dgray001/gray_online/game/games/risq/internal/defs"

func (b *RisqBuilding) autoAttackTarget(risq *GameRisq) Attackable {
	if space_radius, ranged := b.attack_range.SpaceRadius(); ranged {
		return nearbyAttackTarget(b.zone, b.player_id, b.target_priority, b.inAttackRange, risq, space_radius)
	}
	return zoneAttackTarget(b.zone, b.player_id, b.target_priority)
}

func (b *RisqBuilding) autoAttackOrder(risq *GameRisq, order_type defs.OrderType, target_id int64) {
	active := b.order_queue.active_orders
	if len(active) > 0 && active[0].order_type == order_type && active[0].target_id == target_id {
		return
	}
	order := createRisqOrder(risq.nextOrderInternalId(), order_type, b.player_id, map[uint64]Orderable{b.internal_id: b}, target_id, false)
	if !b.orderReceivable(order, risq) {
		return
	}
	if len(active) > 0 && active[0].order_type.IsAutoSynthesized() {
		active = active[1:]
	}
	b.order_queue.past_orders = append(b.order_queue.past_orders, order)
	b.order_queue.active_orders = append([]*RisqOrder{order}, active...)
}

func (b *RisqBuilding) resolveAutoAttack(risq *GameRisq) {
	if !b.auto_attack || b.zone == nil || b.underConstruction() || b.cs.attack_type == defs.AttackType_NONE {
		return
	}
	target := b.autoAttackTarget(risq)
	if target != nil && b.cs.totalAttack() <= 0 && len(b.buildGarrisonAttacks(risq, target)) == 0 {
		if active := b.order_queue.active_orders; len(active) > 0 && active[0].order_type.IsAutoSynthesized() {
			b.order_queue.active_orders = active[1:]
		}
		return
	}
	if target == nil || (!b.interrupt_current && len(b.order_queue.active_orders) > 0) {
		return
	}
	order_type := attackOrderType(target, defs.OrderType_BuildingAutoAttackUnit, defs.OrderType_BuildingAutoAttackBuilding)
	b.autoAttackOrder(risq, order_type, int64(target.internalId()))
}

func (b *RisqBuilding) setBuildingAttackIntent(risq *GameRisq, target Attackable) {
	garrison_attacks := b.buildGarrisonAttacks(risq, target)
	if b.cs.totalAttack() <= 0 && len(garrison_attacks) == 0 {
		return
	}
	b.intent.setBuildingAttack(target, garrison_attacks)
}

func (b *RisqBuilding) buildGarrisonAttacks(risq *GameRisq, target Attackable) []GarrisonAttack {
	config := defs.BuildingConfigs[b.building_id]
	attacks := make([]GarrisonAttack, 0, len(b.garrisoned_units))
	for _, unit := range b.garrisoned_units {
		if unit.deleted || unit.current_stamina <= 0 {
			continue
		}
		stats := risq.effectiveCombatStats(unit, target, true)
		stats.attack_blunt = cappedGarrisonAttack(stats.attack_blunt, attackTypeHasBlunt(b.cs.attack_type), b.cs.attack_blunt, config.Max_garrison_attack_blunt)
		stats.attack_piercing = cappedGarrisonAttack(stats.attack_piercing, attackTypeHasPiercing(b.cs.attack_type), b.cs.attack_piercing, config.Max_garrison_attack_piercing)
		stats.attack_magic = cappedGarrisonAttack(stats.attack_magic, attackTypeHasMagic(b.cs.attack_type), 0, config.Max_garrison_attack_magic)
		if stats.totalAttack() <= 0 {
			continue
		}
		cost := min(unit.current_stamina, unitTickStaminaCost)
		attacks = append(attacks, GarrisonAttack{unit: unit, stats: stats, cost: cost})
	}
	return attacks
}

func cappedGarrisonAttack(unit_attack int, building_has_type bool, building_attack int, override *int) int {
	if !building_has_type {
		unit_attack /= 2
	}
	max_allowed := building_attack
	if override != nil {
		max_allowed = *override
	}
	return min(unit_attack, max_allowed)
}
