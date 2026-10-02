package risq

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

type RisqBuilding struct {
	orderableBase
	building_id        uint32
	display_name       string
	population_support uint16
	garrison_capacity  uint16
	production_queue   map[uint64]*RisqBuildingProductionItem
	// build stamina still needed to finish a unit-constructed foundation; 0 means not under construction
	stamina_remaining          int
	construction_stamina_total int
	health_synced_stamina      int
	garrisoned_units           map[uint64]*RisqUnit
	gather_point               *RisqGatherPoint
	auto_attack                bool
	// Building gathering fields
	resources_left float64
	renewing       *defs.RisqResourceCost
	pending_renew  float64
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
		orderableBase: orderableBase{
			internal_id:     internal_id,
			player_id:       player_id,
			turn_stamina:    10,
			cs:              createRisqCombatStats(),
			order_queue:     createRisqOrderQueue(),
			intent:          createRisqIntent(),
			target_priority: []defs.TargetCategory{},
		},
		building_id:      building_id,
		production_queue: make(map[uint64]*RisqBuildingProductionItem),
		garrisoned_units: make(map[uint64]*RisqUnit),
		auto_attack:      true,
	}
	config, ok := defs.BuildingConfigs[building_id]
	if !ok {
		fmt.Fprintln(os.Stderr, "Creating unknown building id: ", building_id)
		return &building
	}
	building.display_name = config.Display_name
	building.cs.setMaxHealth(config.Max_health)
	building.population_support = config.Population_support
	building.garrison_capacity = config.Garrison_capacity
	building.turn_stamina = config.Turn_stamina
	building.cs.attack_type = config.Attack_type
	building.cs.attack_blunt = config.Attack_blunt
	building.cs.attack_piercing = config.Attack_piercing
	building.cs.defense_blunt = config.Defense_blunt
	building.cs.defense_piercing = config.Defense_piercing
	building.cs.penetration_blunt = config.Penetration_blunt
	building.cs.penetration_piercing = config.Penetration_piercing
	building.attack_range = config.Attack_range
	building.resources_left = config.Gather.Starting_resources
	return &building
}

func (b *RisqBuilding) vision() *defs.RisqVision {
	v := defs.BuildingConfigs[b.building_id].Vision
	return &v
}

func (b *RisqBuilding) score() uint {
	return defs.BuildingConfigs[b.building_id].Cost.Points()
}

func (b *RisqBuilding) OrderableType() defs.OrderableType {
	return defs.OrderableType_BUILDING
}

func (b *RisqBuilding) combatStats(r *GameRisq, other Orderable, attacking bool) RisqCombatStats {
	return b.cs
}

func (b *RisqBuilding) resolveHealthDelta(r *GameRisq) {
	if b.health_synced_stamina != b.stamina_remaining {
		old_ratio := constructionHealthRatio(b.health_synced_stamina, b.construction_stamina_total)
		new_ratio := constructionHealthRatio(b.stamina_remaining, b.construction_stamina_total)
		b.cs.queueHealth(float64(b.cs.max_health) * (new_ratio - old_ratio))
		b.health_synced_stamina = b.stamina_remaining
	}
	if b.cs.pending_health_delta == 0 {
		return
	}
	was_alive := b.isAlive()
	util.DebugLog.Printf("health turn=%d tick=%d: %d b%d before=%.4f delta=%.4f", r.turn_number, r.current_tick, b.internal_id, b.building_id, b.cs.health, b.cs.pending_health_delta)
	b.cs.addHealth(b.cs.pending_health_delta)
	b.cs.pending_health_delta = 0
	if was_alive && !b.isAlive() {
		if event, ok := lowestIdAttackerEvent(b.attacked_by, r.current_tick); ok {
			if attacker := r.resolveAttacker(event); attacker != nil {
				b.recordDeath(r, attacker, event.damage)
			}
		}
	}
}

func (b *RisqBuilding) recordDeath(r *GameRisq, attacker Attackable, damage float64) {
	space := b.zone.space.coordinate
	zone := b.zone.coordinate
	r.players[attacker.playerId()].report.recordCombat(RisqCombatEvent{tick: r.current_tick, kind: RisqCombatEventKind_BUILDING_RAZED,
		self_player: attacker.playerId(), other_player: b.player_id, target_id: uint64(b.building_id), space: space, zone: zone, damage: damage})
	r.players[b.player_id].report.recordCombat(RisqCombatEvent{tick: r.current_tick, kind: RisqCombatEventKind_BUILDING_LOST,
		self_player: b.player_id, other_player: attacker.playerId(), target_id: uint64(b.building_id), space: space, zone: zone, damage: damage})
	r.players[attacker.playerId()].razes++
	if !b.deleted {
		r.players[b.player_id].buildings_lost++
	}
	b.deleted = true
}

func (b *RisqBuilding) cleanupDeleted(risq *GameRisq) {
	player := risq.players[b.player_id]
	for _, item := range b.production_queue {
		player.resources.refund(item.cost)
		if item.kind == defs.ProducibleKind_TECH {
			delete(player.researched_techs, item.item_id)
		}
	}
	for _, unit := range b.garrisoned_units {
		unit.garrisoned_in = nil
		if !unit.deleted && b.zone != nil && b.zone.space != nil {
			b.zone.space.setUnit(&b.zone.coordinate, unit)
		}
	}
	b.garrisoned_units = make(map[uint64]*RisqUnit)
	resolveOrdersOnDeath(risq, &b.order_queue, b.internal_id, defs.OrderType_BuildingDelete)
	if b.zone != nil {
		risq.pending_terrain_clears = append(risq.pending_terrain_clears, b.zone)
	}
	if b.zone != nil && b.zone.space != nil {
		b.zone.space.removeBuilding(b)
	}
	delete(player.buildings, b.internal_id)
	delete(risq.buildings, b.internal_id)
}

type RisqBuildingProductionItem struct {
	kind              defs.ProducibleKind
	item_id           uint32
	stamina_remaining int
	cost              defs.RisqResourceCost
}

func (item *RisqBuildingProductionItem) toFrontend() gin.H {
	return gin.H{
		"kind":              item.kind,
		"item_id":           item.item_id,
		"stamina_remaining": item.stamina_remaining,
	}
}

func (b *RisqBuilding) orderReceivable(o *RisqOrder, risq *GameRisq) bool {
	if o.order_type == defs.OrderType_BuildingDelete {
		return true
	}
	if o.order_type == defs.OrderType_BuildingResearch {
		tech_id := uint32(o.target_id)
		if _, exists := risq.players[b.player_id].researched_techs[tech_id]; exists {
			return false
		}
	}
	if b.underConstruction() {
		return false
	}
	switch o.order_type {
	case defs.OrderType_BuildingAttackUnit, defs.OrderType_BuildingAutoAttackUnit:
		target := risq.units[uint64(o.target_id)]
		return target != nil && target.zone != nil && canAttack(b.player_id, target.player_id) && target.zone.space.getVisibility(b.player_id) >= defs.VisibilityGood
	case defs.OrderType_BuildingAttackBuilding, defs.OrderType_BuildingAutoAttackBuilding:
		target := risq.buildings[uint64(o.target_id)]
		if target == nil || !canAttack(b.player_id, target.player_id) {
			return false
		}
		_, ok := target.zone.buildingKnownTo(b.player_id)
		return ok
	}
	return true
}

func (b *RisqBuilding) receiveOrder(o *RisqOrder, risq *GameRisq, prepend bool) error {
	if active := b.order_queue.active_orders; o.order_type.IsAutoSynthesized() && len(active) > 0 && active[0].order_type.IsAutoSynthesized() {
		b.cancelOrder(active[0], risq)
	}
	switch o.order_type {
	case defs.OrderType_BuildingCreate:
		unit_id := uint32(o.target_id)
		cost, stamina_required := defs.UnitProductionCost(unit_id)
		resources := risq.players[b.player_id].resources
		if !resources.canAfford(cost) {
			return errors.New("cannot afford unit")
		}
		resources.spend(cost)
		b.order_queue.receiveOrder(o, prepend)
		b.production_queue[o.internal_id] = &RisqBuildingProductionItem{
			kind:              defs.ProducibleKind_UNIT,
			item_id:           unit_id,
			stamina_remaining: stamina_required,
			cost:              cost,
		}
	case defs.OrderType_BuildingResearch:
		tech_id := uint32(o.target_id)
		tech := defs.TechConfigs[tech_id]
		resources := risq.players[b.player_id].resources
		if !resources.canAfford(tech.Cost) {
			return errors.New("cannot afford research")
		}
		resources.spend(tech.Cost)
		b.order_queue.receiveOrder(o, prepend)
		b.production_queue[o.internal_id] = &RisqBuildingProductionItem{
			kind:              defs.ProducibleKind_TECH,
			item_id:           tech_id,
			stamina_remaining: tech.Research_stamina,
			cost:              tech.Cost,
		}
		risq.players[b.player_id].researched_techs[tech_id] = false
	default:
		b.order_queue.receiveOrder(o, prepend)
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
	if item.kind == defs.ProducibleKind_TECH {
		delete(risq.players[b.player_id].researched_techs, item.item_id)
	}
}

func (b *RisqBuilding) orderStatus(o *RisqOrder, risq *GameRisq) OrderStatus {
	if o.order_type.IsAutoSynthesized() && !b.auto_attack {
		return OrderStatus_Cancelled
	}
	switch o.order_type {
	case defs.OrderType_BuildingCreate:
		if item, ok := b.production_queue[o.internal_id]; ok && item.stamina_remaining > 0 {
			return OrderStatus_InProgress
		}
	case defs.OrderType_BuildingResearch:
		if item, ok := b.production_queue[o.internal_id]; ok && item.stamina_remaining > 0 {
			return OrderStatus_InProgress
		}
	case defs.OrderType_BuildingDelete:
		if !b.deleted {
			return OrderStatus_InProgress
		}
	case defs.OrderType_BuildingAttackUnit, defs.OrderType_BuildingAutoAttackUnit:
		target := risq.units[uint64(o.target_id)]
		if target == nil || target.isDeleted() {
			break
		}
		if !canAttack(b.player_id, target.player_id) {
			return OrderStatus_Cancelled
		}
		if target.zone == nil || target.zone.space.getVisibility(b.player_id) < defs.VisibilityGood {
			return OrderStatus_Cancelled
		}
		if !b.inAttackRange(target.zone) {
			return OrderStatus_Cancelled
		}
		return OrderStatus_InProgress
	case defs.OrderType_BuildingAttackBuilding, defs.OrderType_BuildingAutoAttackBuilding:
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
		if !b.inAttackRange(target.zone) {
			return OrderStatus_Cancelled
		}
		return OrderStatus_InProgress
	}
	return OrderStatus_Executed
}

func (b *RisqBuilding) tickIntent(risq *GameRisq) bool {
	b.intent.resetIntent()
	b.resolveAutoAttack(risq)
	order := b.order_queue.nextOrder(b, risq)
	if order == nil {
		return false
	}
	switch order.order_type {
	case defs.OrderType_BuildingCreate:
		b.intent.setProduction(order.internal_id, b.production_queue[order.internal_id])
	case defs.OrderType_BuildingResearch:
		b.intent.setProduction(order.internal_id, b.production_queue[order.internal_id])
	case defs.OrderType_BuildingDelete:
		b.intent.setDelete()
	case defs.OrderType_BuildingAttackUnit, defs.OrderType_BuildingAutoAttackUnit:
		target := risq.units[uint64(order.target_id)]
		if b.inAttackRange(target.zone) {
			b.setBuildingAttackIntent(risq, target)
		}
	case defs.OrderType_BuildingAttackBuilding, defs.OrderType_BuildingAutoAttackBuilding:
		target := risq.buildings[uint64(order.target_id)]
		if b.inAttackRange(target.zone) {
			b.setBuildingAttackIntent(risq, target)
		}
	default:
		fmt.Fprintln(os.Stderr, "Order type not implemented:", order.order_type)
	}
	b.intent.resolveCost(b.current_stamina)
	if production, ok := b.intent.detail.(*ProductionIntent); ok && production.item.kind == defs.ProducibleKind_UNIT && risq.players[b.player_id].populationCapped() {
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
		if item.kind == defs.ProducibleKind_UNIT {
			completing_this_tick := item.stamina_remaining-b.intent.intent_cost <= 0
			if completing_this_tick && !risq.population_slot_winners[b] {
				return
			}
			if !completing_this_tick && risq.population_capped[b.player_id] {
				return
			}
		}
		item.stamina_remaining -= b.intent.intent_cost
		if item.stamina_remaining <= 0 {
			switch item.kind {
			case defs.ProducibleKind_UNIT:
				unit := createRisqUnit(risq.unit_creation_ids[b], item.item_id, risq.players[b.player_id])
				unit.current_stamina = unit.turn_stamina / 2
				util.DebugLog.Printf("Unit created: unit=%d unit_id=%d player=%d building=%d zone=%s space=%s tick=%d",
					unit.internal_id, item.item_id, b.player_id, b.internal_id, b.zone.coordinate.ToString(), b.zone.space.coordinate.ToString(), risq.current_tick)
				b.zone.space.setUnit(&b.zone.coordinate, unit)
				risq.players[b.player_id].units[unit.internal_id] = unit
				risq.units[unit.internal_id] = unit
				risq.players[b.player_id].report.recordUnitCreated(item.item_id)
			case defs.ProducibleKind_TECH:
				risq.pending_tech_completions = append(risq.pending_tech_completions, techCompletion{player_id: b.player_id, tech_id: item.item_id})
			}
			delete(b.production_queue, detail.order_internal_id)
		}
	}
	if _, ok := b.intent.detail.(*DeleteIntent); ok {
		b.deleted = true
		risq.players[b.player_id].buildings_lost++
		// TODO: check for recent damage to assign raze
	}
	if detail, ok := b.intent.detail.(*BuildingAttackIntent); ok {
		if b.cs.totalAttack() > 0 && b.intent.intent_cost > 0 {
			risq.buildingAttack(b, detail.target)
		}
		for _, ga := range detail.garrison_attacks {
			// Skip a unit that ended up needing its own stamina this tick (e.g. to ungarrison),
			// so it isn't double-spent between its own intent and this garrison attack.
			if ga.unit.intent.hasIntent() {
				continue
			}
			risq.resolveAttack(garrisonAttacker{RisqUnit: ga.unit, stats: ga.stats}, detail.target, ga.cost)
			ga.unit.current_stamina -= ga.cost
		}
	}
	b.current_stamina -= b.intent.intent_cost
}

func buildingProducesToFrontend(building_id uint32) []gin.H {
	produces := make([]gin.H, 0)
	for _, p := range defs.BuildingConfigs[building_id].Produces {
		produces = append(produces, p.ToFrontend())
	}
	return produces
}

func (b *RisqBuilding) toFrontend(viewer_player_id int) gin.H {
	config := defs.BuildingConfigs[b.building_id]
	display_name := b.display_name
	if config.IsGatherable() && b.resources_left <= 0 {
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
	if showOrdersTo(b.player_id, b.zone, viewer_player_id) {
		building["renewing"] = b.renewing != nil
		building["auto_attack"] = b.auto_attack
		building["interrupt_current"] = b.interrupt_current
		target_priority := make([]int, len(b.target_priority))
		for i, cat := range b.target_priority {
			target_priority[i] = int(cat)
		}
		building["target_priority"] = target_priority
	}
	if b.gather_point != nil && showOrdersTo(b.player_id, b.zone, viewer_player_id) {
		building["gather_point"] = b.gather_point.toFrontend()
	}
	building["produces"] = buildingProducesToFrontend(b.building_id)
	if config.IsGatherable() {
		building["resources_left"] = b.resources_left
		building["gather_capacity"] = config.Gather.Gather_capacity
		building["base_gather_speed"] = config.Gather.Base_gather_speed
		building["renew_cost"] = config.Gather.Renew_cost.ToFrontend()
		building["resource_category"] = config.Gather.Resource_category
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

func AllBuildingConfigsToFrontend() []gin.H {
	ids := make([]uint32, 0, len(defs.BuildingConfigs))
	for id := range defs.BuildingConfigs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	buildings := make([]gin.H, len(ids))
	for i, id := range ids {
		config := defs.BuildingConfigs[id]
		produces := make([]gin.H, len(config.Produces))
		for j, p := range config.Produces {
			produces[j] = p.ToFrontend()
		}
		buildings[i] = gin.H{
			"building_id":      id,
			"display_name":     config.Display_name,
			"description":      config.Description,
			"required_tech_id": config.Required_tech_id,
			"produces":         produces,
			"cost":             config.Cost.ToFrontend(),
			"stamina_cost":     config.Build_stamina,
			"stats": gin.H{
				"health":               config.Max_health,
				"attack_type":          config.Attack_type,
				"attack_blunt":         config.Attack_blunt,
				"attack_piercing":      config.Attack_piercing,
				"attack_range":         config.Attack_range,
				"defense_blunt":        config.Defense_blunt,
				"defense_piercing":     config.Defense_piercing,
				"penetration_blunt":    config.Penetration_blunt,
				"penetration_piercing": config.Penetration_piercing,
			},
		}
	}
	return buildings
}
