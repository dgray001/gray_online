package risq

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

type RisqUnit struct {
	deleted         bool
	internal_id     uint64
	player_id       int
	unit_id         uint32
	display_name    string
	zone            *RisqZone
	turn_stamina    int
	current_stamina int
	cs              RisqCombatStats
	order_queue     RisqOrderQueue
	intent          *RisqIntent
}

func createRisqUnit(internal_id uint64, unit_id uint32, player *RisqPlayer) *RisqUnit {
	unit := RisqUnit{
		deleted:         false,
		internal_id:     internal_id,
		player_id:       player.player.Player_id,
		unit_id:         unit_id,
		turn_stamina:    10,
		current_stamina: 0,
		cs:              createRisqCombatStats(),
		order_queue:     createRisqOrderQueue(),
		intent:          createRisqIntent(),
	}
	config, ok := unitConfigs[unit_id]
	if !ok {
		fmt.Fprintln(os.Stderr, "Creating unknown unit id: ", unit_id)
		return &unit
	}
	unit.display_name = config.display_name
	unit.cs.setMaxHealth(config.max_health)
	unit.cs.attack_type = config.attack_type
	unit.cs.attack_blunt = config.attack_blunt
	unit.cs.attack_piercing = config.attack_piercing
	unit.cs.defense_blunt = config.defense_blunt
	unit.cs.defense_piercing = config.defense_piercing
	unit.cs.penetration_blunt = config.penetration_blunt
	unit.cs.penetration_piercing = config.penetration_piercing
	unit.turn_stamina = config.turn_stamina
	for tech_id, researched := range player.researched_techs {
		if !researched {
			continue
		}
		tech, ok := techConfigs[tech_id]
		if !ok || tech.affects_unit_id != unit_id {
			continue
		}
		applyTechBonus(&unit, tech)
	}
	return &unit
}

func unitProductionCost(unit_id uint32) (RisqResourceCost, int) {
	config, ok := unitConfigs[unit_id]
	if !ok {
		fmt.Fprintln(os.Stderr, "Unknown unit id for production cost: ", unit_id)
		return RisqResourceCost{}, 0
	}
	return config.cost, config.production_stamina
}

func (u *RisqUnit) vision() *RisqVision {
	v := unitConfigs[u.unit_id].vision
	return &v
}

func (u *RisqUnit) score() uint {
	return unitConfigs[u.unit_id].cost.points()
}

func (u *RisqUnit) isDeleted() bool {
	return u.deleted
}

func (u *RisqUnit) internalId() uint64 {
	return u.internal_id
}

func (u *RisqUnit) cleanupDeleted(risq *GameRisq) {
	resolveOrdersOnDeath(risq, &u.order_queue, u.internal_id, OrderType_UnitDelete)
	if u.zone != nil && u.zone.space != nil {
		u.zone.space.removeUnit(u)
	}
	delete(risq.players[u.player_id].units, u.internal_id)
	delete(risq.units, u.internal_id)
}

func (u *RisqUnit) refreshStamina() {
	u.current_stamina += u.turn_stamina
	max_stamina := maxStaminaFor(u.turn_stamina)
	if u.current_stamina > max_stamina {
		u.current_stamina = max_stamina
	}
}

func (u *RisqUnit) receiveOrder(o *RisqOrder, risq *GameRisq) {
	u.order_queue.receiveOrder(o)
	switch o.order_type {
	case OrderType_UnitBuild:
		building_id, _, zone := invertBuildKey(uint(o.target_id), risq)
		player := risq.players[u.player_id]
		if zone.building != nil || player.planned_foundations[zone.coordinate_key] != nil {
			return
		}
		cost, _ := buildingProductionCost(building_id)
		if !player.resources.canAfford(cost) {
			player.report.recordFailure(o.order_type, o.target_id, "cannot afford building")
			return
		}
		player.planned_foundations[zone.coordinate_key] = createRisqPlannedFoundation(building_id, player)
	}
}

func (u *RisqUnit) cancelOrder(o *RisqOrder, risq *GameRisq) {
	u.order_queue.removeOrder(o.internal_id)
	if len(o.subjects) > 1 {
		delete(o.subjects, u.internal_id)
	} else {
		o.cancelled = true
		o.turn_resolved = risq.turn_number
	}
	switch o.order_type {
	case OrderType_UnitBuild:
		// the planned foundation is independent of the order that created it; cancelling the order doesn't touch it
	}
}

func (u *RisqUnit) orderReceivable(o *RisqOrder, risq *GameRisq) bool {
	switch o.order_type {
	case OrderType_UnitBuild:
		building_id, _, zone := invertBuildKey(uint(o.target_id), risq)
		if owner := zone.space.ownership; owner >= 0 && owner != u.player_id {
			return false
		}
		if zone.resource != nil {
			return false
		}
		if zone.building != nil {
			b := zone.building
			return b.player_id == u.player_id && b.underConstruction() && b.building_id == building_id
		}
		foundation := risq.players[u.player_id].planned_foundations[zone.coordinate_key]
		return foundation == nil || foundation.building_id == building_id
	case OrderType_UnitAttackUnit:
		target := risq.units[uint64(o.target_id)]
		return target != nil && canAttack(u.player_id, target.player_id)
	case OrderType_UnitAttackBuilding:
		target := risq.buildings[uint64(o.target_id)]
		return target != nil && canAttack(u.player_id, target.player_id)
	default:
	}
	return true
}

func (r *GameRisq) canAssist(player_id int, b *RisqBuilding) bool {
	return b.player_id == player_id // TODO: own-or-ally once diplomacy exists
}

func canAttack(attacker_player_id int, target_player_id int) bool {
	return attacker_player_id != target_player_id // TODO: not-ally once diplomacy exists
}

func (u *RisqUnit) orderStatus(o *RisqOrder, risq *GameRisq) OrderStatus {
	switch o.order_type {
	case OrderType_UnitMoveSpace:
		space := invertSpaceKey(uint(o.target_id), risq)
		if u.zone.space != space {
			return OrderStatus_InProgress
		}
	case OrderType_UnitMoveZone:
		_, zone := invertZoneKey(uint(o.target_id), risq)
		if u.zone != zone {
			return OrderStatus_InProgress
		}
	case OrderType_UnitGather:
		_, zone := invertZoneKey(uint(o.target_id), risq)
		if zone.resource != nil && zone.resource.resources_left > 0 {
			return OrderStatus_InProgress
		}
	case OrderType_UnitBuild:
		building_id, _, zone := invertBuildKey(uint(o.target_id), risq)
		if zone.building != nil && zone.building.deleted {
			return OrderStatus_Cancelled
		}
		if zone.building == nil {
			if risq.players[u.player_id].planned_foundations[zone.coordinate_key] == nil {
				return OrderStatus_Cancelled
			}
			owner := zone.space.computeOwnership()
			if owner >= 0 && owner != u.player_id {
				risq.players[u.player_id].cancelPlannedFoundation(zone)
				return OrderStatus_Cancelled
			}
			if u.zone == zone && owner != u.player_id {
				risq.players[u.player_id].cancelPlannedFoundation(zone)
				return OrderStatus_Cancelled
			}
			return OrderStatus_InProgress
		}
		if zone.building.player_id != u.player_id || zone.building.building_id != building_id {
			risq.players[u.player_id].cancelPlannedFoundation(zone)
			return OrderStatus_Cancelled
		}
		if zone.building.underConstruction() {
			return OrderStatus_InProgress
		}
	case OrderType_UnitRepair:
		target := risq.buildings[uint64(o.target_id)]
		if target == nil || target.isDeleted() {
			return OrderStatus_Cancelled
		}
		if !target.underConstruction() && risq.canAssist(u.player_id, target) && target.cs.health < float64(target.cs.max_health) {
			if _, cost, ok := repairHealAndCost(target, 1); !ok || risq.players[u.player_id].resources.affordFraction(cost) <= 0 {
				return OrderStatus_Cancelled
			}
			return OrderStatus_InProgress
		}
	case OrderType_UnitDelete:
		if !u.deleted {
			return OrderStatus_InProgress
		}
	case OrderType_UnitAttackBuilding:
		target := risq.buildings[uint64(o.target_id)]
		if target == nil || target.isDeleted() {
			break
		}
		if !canAttack(u.player_id, target.player_id) {
			return OrderStatus_Cancelled
		}
		return OrderStatus_InProgress
	case OrderType_UnitAttackUnit:
		target := risq.units[uint64(o.target_id)]
		if target == nil || target.isDeleted() {
			break
		}
		if !canAttack(u.player_id, target.player_id) {
			return OrderStatus_Cancelled
		}
		return OrderStatus_InProgress
	case OrderType_UnitAttackZone:
		_, zone := invertZoneKey(uint(o.target_id), risq)
		if u.zone != zone || zoneHasEnemy(zone, u.player_id) {
			return OrderStatus_InProgress
		}
	case OrderType_UnitAttackSpace:
		space := invertSpaceKey(uint(o.target_id), risq)
		if u.zone.space != space || spaceHasEnemy(space, u.player_id) {
			return OrderStatus_InProgress
		}
	}
	return OrderStatus_Executed
}

func (u *RisqUnit) tickIntent(risq *GameRisq) bool {
	u.intent.resetIntent()
	order := u.order_queue.nextOrder(u, risq)
	if order == nil {
		return false
	}
	switch order.order_type {
	case OrderType_UnitMoveSpace:
		space := invertSpaceKey(uint(order.target_id), risq)
		u.intent.setMove(u.findPath(space.getCenterZone()))
	case OrderType_UnitMoveZone:
		_, zone := invertZoneKey(uint(order.target_id), risq)
		u.intent.setMove(u.findPath(zone))
	case OrderType_UnitGather:
		_, zone := invertZoneKey(uint(order.target_id), risq)
		if u.zone != zone {
			u.intent.setMove(u.findPath(zone))
		} else {
			u.intent.setGather(zone.resource)
		}
	case OrderType_UnitBuild:
		building_id, _, zone := invertBuildKey(uint(order.target_id), risq)
		if u.zone != zone {
			u.intent.setMove(u.findPath(zone))
		} else {
			u.intent.setBuild(zone.building, building_id, zone)
		}
	case OrderType_UnitDelete:
		u.intent.setDelete()
	case OrderType_UnitAttackBuilding:
		target := risq.buildings[uint64(order.target_id)]
		if u.zone != target.zone {
			u.intent.setMove(u.findPath(target.zone))
		} else {
			u.intent.setAttackBuilding(target)
		}
	case OrderType_UnitAttackUnit:
		target := risq.units[uint64(order.target_id)]
		if u.zone != target.zone {
			u.intent.setMove(u.findPath(target.zone))
		} else {
			u.intent.setAttackUnit(target)
		}
	case OrderType_UnitAttackZone:
		_, zone := invertZoneKey(uint(order.target_id), risq)
		if u.zone != zone {
			u.intent.setMove(u.findPath(zone))
		} else if target := zoneEnemyUnit(zone, u.player_id); target != nil {
			u.intent.setAttackUnit(target)
		} else if target := zoneEnemyBuilding(zone, u.player_id); target != nil {
			u.intent.setAttackBuilding(target)
		}
	case OrderType_UnitAttackSpace:
		space := invertSpaceKey(uint(order.target_id), risq)
		if u.zone.space != space {
			u.intent.setMove(u.findPath(space.getCenterZone()))
		} else if target_unit, target_building := spaceAttackTarget(u, space); target_unit != nil {
			if u.zone == target_unit.zone {
				u.intent.setAttackUnit(target_unit)
			} else {
				u.intent.setMove(u.findPath(target_unit.zone))
			}
		} else if target_building != nil {
			if u.zone == target_building.zone {
				u.intent.setAttackBuilding(target_building)
			} else {
				u.intent.setMove(u.findPath(target_building.zone))
			}
		}
	case OrderType_UnitRepair:
		target := risq.buildings[uint64(order.target_id)]
		if target == nil {
			break
		}
		if u.zone != target.zone {
			u.intent.setMove(u.findPath(target.zone))
		} else {
			u.intent.setRepair(target)
		}
	default:
		fmt.Fprintln(os.Stderr, "Order type not implemented:", order.order_type)
	}
	u.intent.resolveCost(u.current_stamina)
	return u.intent.hasIntent()
}

func (u *RisqUnit) tickExecute(risq *GameRisq) {
	if !u.intent.hasIntent() {
		return
	}
	switch detail := u.intent.detail.(type) {
	case *MoveIntent:
		fmt.Println("Moving unit", u.display_name, "to zone"+detail.next_step.coordinate.ToString(), "in space", detail.next_step.space.coordinate.ToString())
		old_zone := u.zone
		new_zone := detail.next_step
		if old_zone.space == new_zone.space {
			delete(old_zone.units, u.internal_id)
			u.zone = new_zone
			new_zone.units[u.internal_id] = u
		} else {
			old_zone.space.removeUnit(u)
			new_zone.space.setUnit(&new_zone.coordinate, u)
		}
	case *GatherIntent:
		amount, ok := risq.gather_allotments[u]
		if !ok {
			amount = float64(u.intent.intent_cost) * (float64(detail.resource.base_gather_speed) / gatherRateStaminaBase)
		}
		if amount > detail.resource.resources_left {
			amount = detail.resource.resources_left
		}
		detail.resource.resources_left -= amount
		risq.players[u.player_id].resources.addGathered(detail.resource.category(), amount)
		if detail.resource.resources_left <= 0 && detail.resource.zone != nil {
			if detail.resource.zone.space != nil {
				delete(detail.resource.zone.space.resources, detail.resource.internal_id)
			}
			detail.resource.zone.resource = nil
		}
	case *ConstructionIntent:
		if detail.zone.building != nil && detail.zone.building.player_id != u.player_id {
			return
		}
		building := detail.building_under_construction
		if building == nil {
			if detail.zone.building != nil {
				building = detail.zone.building
			} else {
				if owner := detail.zone.space.computeOwnership(); owner != u.player_id {
					return
				}
				_, stamina_required := buildingProductionCost(detail.building_id)
				building = createRisqBuilding(risq.nextBuildingInternalId(), detail.building_id, u.player_id)
				building.stamina_remaining = stamina_required
				building.construction_stamina_total = stamina_required
				building.cs.setHealthRatio(constructionHealthRatio(stamina_required, stamina_required))
				detail.zone.space.setBuilding(&detail.zone.coordinate, building)
				risq.players[u.player_id].buildings[building.internal_id] = building
				risq.buildings[building.internal_id] = building
				delete(risq.players[u.player_id].planned_foundations, detail.zone.coordinate_key)
			}
		}
		if building.deleted || !building.underConstruction() {
			return
		}
		old_ratio := constructionHealthRatio(building.stamina_remaining, building.construction_stamina_total)
		building.stamina_remaining -= u.intent.intent_cost
		new_ratio := constructionHealthRatio(building.stamina_remaining, building.construction_stamina_total)
		building.cs.addHealth(float64(building.cs.max_health) * (new_ratio - old_ratio))
		if !building.underConstruction() {
			risq.players[building.player_id].report.recordBuildingBuilt(building.building_id, building.zone.space.coordinate, building.zone.coordinate)
		}
	case *DeleteIntent:
		u.deleted = true
	case *AttackBuildingIntent:
		risq.unitAttackBuilding(u, detail.target)
	case *AttackUnitIntent:
		risq.unitAttackUnit(u, detail.target)
	case *RepairIntent:
		building := detail.target
		if building.deleted || building.underConstruction() || !risq.canAssist(u.player_id, building) {
			return
		}
		heal, cost, ok := repairHealAndCost(building, u.intent.intent_cost)
		if !ok {
			return
		}
		player := risq.players[u.player_id]
		afford := player.resources.affordFraction(cost)
		if afford <= 0 {
			return
		}
		building.cs.addHealth(heal * afford)
		player.resources.spend(cost.scale(afford))
	}
	u.current_stamina -= u.intent.intent_cost
	fmt.Println("Unit in zone", u.zone.coordinate.ToString(), "of space", u.zone.space.coordinate.ToString())
}

func (u *RisqUnit) toFrontend(viewer_player_id int) gin.H {
	unit := gin.H{
		"internal_id":     u.internal_id,
		"player_id":       u.player_id,
		"unit_id":         u.unit_id,
		"display_name":    u.display_name,
		"turn_stamina":    u.turn_stamina,
		"current_stamina": u.current_stamina,
		"max_stamina":     maxStaminaFor(u.turn_stamina),
		"combat_stats":    u.cs.toFrontend(),
	}
	builds := make([]gin.H, 0)
	for _, p := range unitConfigs[u.unit_id].builds {
		builds = append(builds, p.toFrontend())
	}
	unit["builds"] = builds
	if u.zone != nil {
		unit["zone_coordinate"] = u.zone.coordinate.ToFrontend()
		if u.zone.space != nil {
			unit["space_coordinate"] = u.zone.space.coordinate.ToFrontend()
		}
	}
	active_orders := make([]gin.H, 0)
	if showOrdersTo(u.player_id, u.zone, viewer_player_id) {
		for _, order := range u.order_queue.active_orders {
			if order != nil && !order.executed {
				active_orders = append(active_orders, order.toFrontend())
			}
		}
	}
	unit["active_orders"] = active_orders
	return unit
}
