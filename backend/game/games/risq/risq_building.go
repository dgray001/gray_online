package risq

import (
	"errors"
	"fmt"
	"os"

	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

type RisqBuilding struct {
	deleted            bool
	internal_id        uint64
	player_id          int
	building_id        uint32
	display_name       string
	zone               *RisqZone
	population_support uint16
	garrison_capacity  uint16
	turn_stamina       int
	current_stamina    int
	cs                 RisqCombatStats
	order_queue        RisqOrderQueue
	production_queue   map[uint64]*RisqBuildingProductionItem
	intent             *RisqIntent
	// build stamina still needed to finish a unit-constructed foundation; 0 means not under construction
	stamina_remaining          int
	construction_stamina_total int
	garrisoned_units           map[uint64]*RisqUnit
	gather_point               *RisqGatherPoint
	attacked_by                []RisqDamageEvent
	auto_attack                bool
	interrupt_current          bool
	target_priority            []TargetCategory
	attack_range               RisqRange
	// Building gathering fields
	resources_left float64
	renewing       *RisqResourceCost
}

func (b *RisqBuilding) underConstruction() bool {
	return b.stamina_remaining > 0
}

const constructionMinHealthRatio = 0.1

func constructionHealthRatio(stamina_remaining int, construction_stamina_total int) float64 {
	if construction_stamina_total <= 0 {
		return 1
	}
	progress := util.Clamp(1-float64(stamina_remaining)/float64(construction_stamina_total), 0.0, 1.0)
	return constructionMinHealthRatio + (1-constructionMinHealthRatio)*progress
}

func createRisqBuilding(internal_id uint64, building_id uint32, player_id int) *RisqBuilding {
	building := RisqBuilding{
		deleted:            false,
		internal_id:        internal_id,
		player_id:          player_id,
		building_id:        building_id,
		population_support: 0,
		garrison_capacity:  0,
		turn_stamina:       10,
		current_stamina:    0,
		cs:                 createRisqCombatStats(),
		order_queue:        createRisqOrderQueue(),
		production_queue:   make(map[uint64]*RisqBuildingProductionItem),
		intent:             createRisqIntent(),
		garrisoned_units:   make(map[uint64]*RisqUnit),
		auto_attack:        true,
		interrupt_current:  false,
		target_priority:    []TargetCategory{},
	}
	config, ok := buildingConfigs[building_id]
	if !ok {
		fmt.Fprintln(os.Stderr, "Creating unknown building id: ", building_id)
		return &building
	}
	building.display_name = config.display_name
	building.cs.setMaxHealth(config.max_health)
	building.population_support = config.population_support
	building.garrison_capacity = config.garrison_capacity
	building.turn_stamina = config.turn_stamina
	building.cs.attack_type = config.attack_type
	building.cs.attack_blunt = config.attack_blunt
	building.cs.attack_piercing = config.attack_piercing
	building.cs.defense_blunt = config.defense_blunt
	building.cs.defense_piercing = config.defense_piercing
	building.cs.penetration_blunt = config.penetration_blunt
	building.cs.penetration_piercing = config.penetration_piercing
	building.attack_range = config.attack_range
	building.resources_left = config.gather.starting_resources
	return &building
}

func (b *RisqBuilding) vision() *RisqVision {
	v := buildingConfigs[b.building_id].vision
	return &v
}

func (b *RisqBuilding) score() uint {
	return buildingConfigs[b.building_id].cost.points()
}

func (b *RisqBuilding) isDeleted() bool {
	return b.deleted
}

func (b *RisqBuilding) internalId() uint64 {
	return b.internal_id
}

func (b *RisqBuilding) OrderableType() OrderableType {
	return OrderableType_BUILDING
}

func (b *RisqBuilding) playerId() int {
	return b.player_id
}

func (b *RisqBuilding) combatStats(r *GameRisq, other Orderable, attacking bool) RisqCombatStats {
	return b.cs
}

func (b *RisqBuilding) isAlive() bool {
	return b.cs.health > 0
}

func (b *RisqBuilding) applyDamage(event RisqDamageEvent) {
	b.cs.addHealth(-event.damage)
	b.attacked_by = append(b.attacked_by, event)
}

func (b *RisqBuilding) recordDeath(r *GameRisq, attacker Attackable, damage float64) {
	space := b.zone.space.coordinate
	zone := b.zone.coordinate
	r.players[attacker.playerId()].report.recordCombat(RisqCombatEvent{tick: r.current_tick, kind: RisqCombatEventKind_BUILDING_RAZED,
		self_player: attacker.playerId(), other_player: b.player_id, target_id: uint64(b.building_id), space: space, zone: zone, damage: damage})
	r.players[b.player_id].report.recordCombat(RisqCombatEvent{tick: r.current_tick, kind: RisqCombatEventKind_BUILDING_LOST,
		self_player: b.player_id, other_player: attacker.playerId(), target_id: uint64(b.building_id), space: space, zone: zone, damage: damage})
	r.players[attacker.playerId()].razes++
	r.players[b.player_id].buildings_lost++
	b.deleted = true
}

func (b *RisqBuilding) activeOrders() []*RisqOrder {
	return b.order_queue.active_orders
}

func (b *RisqBuilding) cleanupDeleted(risq *GameRisq) {
	player := risq.players[b.player_id]
	for _, item := range b.production_queue {
		player.resources.refund(item.cost)
		if item.kind == ProducibleKind_TECH {
			delete(player.researched_techs, item.item_id)
		}
	}
	for _, unit := range b.garrisoned_units {
		unit.garrisoned_in = nil
		if b.zone != nil && b.zone.space != nil {
			b.zone.space.setUnit(&b.zone.coordinate, unit)
		}
	}
	b.garrisoned_units = make(map[uint64]*RisqUnit)
	resolveOrdersOnDeath(risq, &b.order_queue, b.internal_id, OrderType_BuildingDelete)
	if b.zone != nil && b.zone.space != nil {
		b.zone.space.removeBuilding(b)
	}
	delete(player.buildings, b.internal_id)
	delete(risq.buildings, b.internal_id)
}

func (b *RisqBuilding) refreshStamina() {
	b.current_stamina += b.turn_stamina
	max_stamina := maxStaminaFor(b.turn_stamina)
	if b.current_stamina > max_stamina {
		b.current_stamina = max_stamina
	}
	b.attacked_by = nil
}

type RisqBuildingProductionItem struct {
	kind              ProducibleKind
	item_id           uint32
	stamina_remaining int
	cost              RisqResourceCost
}

func (item *RisqBuildingProductionItem) toFrontend() gin.H {
	return gin.H{
		"kind":              item.kind,
		"item_id":           item.item_id,
		"stamina_remaining": item.stamina_remaining,
	}
}

func (b *RisqBuilding) orderReceivable(o *RisqOrder, risq *GameRisq) bool {
	if o.order_type == OrderType_BuildingDelete {
		return true
	}
	if o.order_type == OrderType_BuildingResearch {
		tech_id := uint32(o.target_id)
		if _, exists := risq.players[b.player_id].researched_techs[tech_id]; exists {
			return false
		}
	}
	if b.underConstruction() {
		return false
	}
	switch o.order_type {
	case OrderType_BuildingAttackUnit, OrderType_BuildingAutoAttackUnit:
		target := risq.units[uint64(o.target_id)]
		return target != nil && target.zone != nil && canAttack(b.player_id, target.player_id) && target.zone.space.getVisibility(b.player_id) >= VisibilityGood
	case OrderType_BuildingAttackBuilding, OrderType_BuildingAutoAttackBuilding:
		target := risq.buildings[uint64(o.target_id)]
		if target == nil || !canAttack(b.player_id, target.player_id) {
			return false
		}
		_, ok := target.zone.buildingKnownTo(b.player_id)
		return ok
	}
	return true
}

func (b *RisqBuilding) receiveOrder(o *RisqOrder, risq *GameRisq) error {
	switch o.order_type {
	case OrderType_BuildingCreate:
		unit_id := uint32(o.target_id)
		cost, stamina_required := unitProductionCost(unit_id)
		resources := risq.players[b.player_id].resources
		if !resources.canAfford(cost) {
			return errors.New("cannot afford unit")
		}
		resources.spend(cost)
		b.order_queue.receiveOrder(o)
		b.production_queue[o.internal_id] = &RisqBuildingProductionItem{
			kind:              ProducibleKind_UNIT,
			item_id:           unit_id,
			stamina_remaining: stamina_required,
			cost:              cost,
		}
	case OrderType_BuildingResearch:
		tech_id := uint32(o.target_id)
		tech := techConfigs[tech_id]
		resources := risq.players[b.player_id].resources
		if !resources.canAfford(tech.cost) {
			return errors.New("cannot afford research")
		}
		resources.spend(tech.cost)
		b.order_queue.receiveOrder(o)
		b.production_queue[o.internal_id] = &RisqBuildingProductionItem{
			kind:              ProducibleKind_TECH,
			item_id:           tech_id,
			stamina_remaining: tech.research_stamina,
			cost:              tech.cost,
		}
		risq.players[b.player_id].researched_techs[tech_id] = false
	default:
		b.order_queue.receiveOrder(o)
	}
	return nil
}

func (b *RisqBuilding) cancelOrder(o *RisqOrder, risq *GameRisq) {
	b.order_queue.removeOrder(o.internal_id)
	if len(o.subjects) > 1 {
		delete(o.subjects, b.internal_id)
	} else {
		o.cancelled = true
		o.turn_resolved = risq.turn_number
	}
	item, ok := b.production_queue[o.internal_id]
	if !ok {
		return
	}
	risq.players[b.player_id].resources.refund(item.cost)
	delete(b.production_queue, o.internal_id)
	if item.kind == ProducibleKind_TECH {
		delete(risq.players[b.player_id].researched_techs, item.item_id)
	}
}

func (b *RisqBuilding) orderStatus(o *RisqOrder, risq *GameRisq) OrderStatus {
	switch o.order_type {
	case OrderType_BuildingCreate:
		if item, ok := b.production_queue[o.internal_id]; ok && item.stamina_remaining > 0 {
			return OrderStatus_InProgress
		}
	case OrderType_BuildingResearch:
		if item, ok := b.production_queue[o.internal_id]; ok && item.stamina_remaining > 0 {
			return OrderStatus_InProgress
		}
	case OrderType_BuildingDelete:
		if !b.deleted {
			return OrderStatus_InProgress
		}
	case OrderType_BuildingAttackUnit, OrderType_BuildingAutoAttackUnit:
		target := risq.units[uint64(o.target_id)]
		if target == nil || target.isDeleted() {
			break
		}
		if !canAttack(b.player_id, target.player_id) {
			return OrderStatus_Cancelled
		}
		if target.zone == nil || target.zone.space.getVisibility(b.player_id) < VisibilityGood {
			return OrderStatus_Cancelled
		}
		return OrderStatus_InProgress
	case OrderType_BuildingAttackBuilding, OrderType_BuildingAutoAttackBuilding:
		target := risq.buildings[uint64(o.target_id)]
		if target == nil || target.isDeleted() {
			break
		}
		if !canAttack(b.player_id, target.player_id) {
			return OrderStatus_Cancelled
		}
		if _, ok := target.zone.buildingKnownTo(b.player_id); !ok {
			return OrderStatus_Cancelled
		}
		return OrderStatus_InProgress
	}
	return OrderStatus_Executed
}

func (b *RisqBuilding) autoAttackTarget(risq *GameRisq) (*RisqUnit, *RisqBuilding) {
	if space_radius, ranged := b.attack_range.spaceRadius(); ranged {
		return nearbyAttackTarget(b.zone, b.player_id, b.target_priority, b.inAttackRange, risq, space_radius)
	}
	return zoneAttackTarget(b.zone, b.player_id, b.target_priority)
}

func (b *RisqBuilding) autoAttackOrder(risq *GameRisq, order_type OrderType, target_id int64) {
	active := b.order_queue.active_orders
	if len(active) > 0 && active[0].order_type == order_type && active[0].target_id == target_id {
		return
	}
	order := createRisqOrder(risq.nextOrderInternalId(), order_type, b.player_id, map[uint64]Orderable{b.internal_id: b}, target_id, false)
	if !b.orderReceivable(order, risq) {
		return
	}
	if len(active) > 0 && active[0].order_type.isAutoSynthesized() {
		active = active[1:]
	}
	b.order_queue.past_orders = append(b.order_queue.past_orders, order)
	b.order_queue.active_orders = append([]*RisqOrder{order}, active...)
}

func (b *RisqBuilding) resolveAutoAttack(risq *GameRisq) {
	if !b.auto_attack || b.zone == nil || b.underConstruction() {
		return
	}
	if !b.interrupt_current && len(b.order_queue.active_orders) > 0 {
		return
	}
	target, target_building := b.autoAttackTarget(risq)
	if target != nil {
		b.autoAttackOrder(risq, OrderType_BuildingAutoAttackUnit, int64(target.internal_id))
	} else if target_building != nil {
		b.autoAttackOrder(risq, OrderType_BuildingAutoAttackBuilding, int64(target_building.internal_id))
	}
}

func (b *RisqBuilding) tickIntent(risq *GameRisq) bool {
	b.intent.resetIntent()
	b.resolveAutoAttack(risq)
	order := b.order_queue.nextOrder(b, risq)
	if order == nil {
		return false
	}
	switch order.order_type {
	case OrderType_BuildingCreate:
		b.intent.setProduction(order.internal_id, b.production_queue[order.internal_id])
	case OrderType_BuildingResearch:
		b.intent.setProduction(order.internal_id, b.production_queue[order.internal_id])
	case OrderType_BuildingDelete:
		b.intent.setDelete()
	case OrderType_BuildingAttackUnit, OrderType_BuildingAutoAttackUnit:
		target := risq.units[uint64(order.target_id)]
		if b.inAttackRange(target.zone) {
			b.intent.setAttackUnit(target)
		}
	case OrderType_BuildingAttackBuilding, OrderType_BuildingAutoAttackBuilding:
		target := risq.buildings[uint64(order.target_id)]
		if b.inAttackRange(target.zone) {
			b.intent.setAttackBuilding(target)
		}
	default:
		fmt.Fprintln(os.Stderr, "Order type not implemented:", order.order_type)
	}
	b.intent.resolveCost(b.current_stamina)
	if production, ok := b.intent.detail.(*ProductionIntent); ok && production.item.kind == ProducibleKind_UNIT && risq.players[b.player_id].populationCapped() {
		b.intent.resetIntent()
	}
	return b.intent.hasIntent()
}

func (b *RisqBuilding) tickExecute(risq *GameRisq) {
	if !b.intent.hasIntent() {
		return
	}
	if detail, ok := b.intent.detail.(*ProductionIntent); ok {
		item := detail.item
		if item.kind == ProducibleKind_UNIT && risq.players[b.player_id].populationCapped() {
			return
		}
		item.stamina_remaining -= b.intent.intent_cost
		if item.stamina_remaining <= 0 {
			switch item.kind {
			case ProducibleKind_UNIT:
				unit := createRisqUnit(risq.nextUnitInternalId(), item.item_id, risq.players[b.player_id])
				b.zone.space.setUnit(&b.zone.coordinate, unit)
				risq.players[b.player_id].units[unit.internal_id] = unit
				risq.units[unit.internal_id] = unit
				risq.players[b.player_id].report.recordUnitCreated(item.item_id)
				if b.gather_point != nil {
					unit.receiveOrder(b.gather_point.resolveOrder(risq, b, unit), risq)
				}
			case ProducibleKind_TECH:
				risq.completeResearch(risq.players[b.player_id], item.item_id)
			}
			delete(b.production_queue, detail.order_internal_id)
		}
	}
	if _, ok := b.intent.detail.(*DeleteIntent); ok {
		b.deleted = true
		risq.players[b.player_id].buildings_lost++
		// TODO: check for recent damage to assign raze
	}
	if detail, ok := b.intent.detail.(*AttackUnitIntent); ok {
		risq.buildingAttackUnit(b, detail.target)
	}
	if detail, ok := b.intent.detail.(*AttackBuildingIntent); ok {
		risq.buildingAttackBuilding(b, detail.target)
	}
	b.current_stamina -= b.intent.intent_cost
}

func buildingProducesToFrontend(building_id uint32) []gin.H {
	produces := make([]gin.H, 0)
	for _, p := range buildingConfigs[building_id].produces {
		produces = append(produces, p.toFrontend())
	}
	return produces
}

func (b *RisqBuilding) toFrontend(viewer_player_id int) gin.H {
	config := buildingConfigs[b.building_id]
	display_name := b.display_name
	if config.isGatherable() && b.resources_left <= 0 {
		display_name += " (expired)"
	}
	building := gin.H{
		"internal_id":                b.internal_id,
		"player_id":                  b.player_id,
		"building_id":                b.building_id,
		"display_name":               display_name,
		"population_support":         b.population_support,
		"combat_stats":               b.cs.toFrontend(),
		"under_construction":         b.underConstruction(),
		"stamina_remaining":          b.stamina_remaining,
		"construction_stamina_total": b.construction_stamina_total,
		"turn_stamina":               b.turn_stamina,
		"current_stamina":            b.current_stamina,
		"max_stamina":                maxStaminaFor(b.turn_stamina),
		"garrison_capacity":          b.garrison_capacity,
		"attack_range":               b.attack_range,
	}
	building["has_garrisoned_units"] = len(b.garrisoned_units) > 0
	if showOrdersTo(b.player_id, b.zone, viewer_player_id) {
		garrisoned_units := make([]uint64, 0)
		for id := range b.garrisoned_units {
			garrisoned_units = append(garrisoned_units, id)
		}
		building["garrisoned_units"] = garrisoned_units
	}
	if b.gather_point != nil && showOrdersTo(b.player_id, b.zone, viewer_player_id) {
		building["gather_point"] = b.gather_point.toFrontend()
	}
	building["produces"] = buildingProducesToFrontend(b.building_id)
	if config.isGatherable() {
		building["resources_left"] = b.resources_left
		building["gather_capacity"] = config.gather.gather_capacity
		building["renew_cost"] = config.gather.renew_cost.toFrontend()
		building["resource_category"] = config.gather.resource_category
	}
	if b.zone != nil {
		building["zone_coordinate"] = b.zone.coordinate.ToFrontend()
		if b.zone.space != nil {
			building["space_coordinate"] = b.zone.space.coordinate.ToFrontend()
		}
	}
	active_orders := make([]gin.H, 0)
	production_queue := make([]gin.H, 0)
	if showOrdersTo(b.player_id, b.zone, viewer_player_id) {
		for _, order := range b.order_queue.active_orders {
			if order == nil || order.executed {
				continue
			}
			active_orders = append(active_orders, order.toFrontend())
			if item, ok := b.production_queue[order.internal_id]; ok {
				item_frontend := item.toFrontend()
				item_frontend["order_internal_id"] = order.internal_id
				production_queue = append(production_queue, item_frontend)
			}
		}
	}
	building["active_orders"] = active_orders
	building["production_queue"] = production_queue
	return building
}
