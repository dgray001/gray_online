package risq

import (
	"errors"
	"fmt"
	"math"
	"os"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

type UnitStance uint8

const (
	UnitStance_NONE UnitStance = iota
	UnitStance_PASSIVE
	UnitStance_AGGRESSIVE
	UnitStance_DEFENSIVE
	UnitStance_STAND_GROUND
	UnitStance_END
)

type UnitType uint8

const (
	UnitType_NONE UnitType = iota
	UnitType_ECONOMIC
	UnitType_INFANTRY
	UnitType_ARCHER
	UnitType_CAVALRY
)

type RisqUnit struct {
	orderableBase
	unit_id       uint32
	display_name  string
	garrisoned_in *RisqBuilding
	stance        UnitStance
	attack_back   bool
}

func createRisqUnit(internal_id uint64, unit_id uint32, player *RisqPlayer) *RisqUnit {
	unit := RisqUnit{
		orderableBase: orderableBase{
			internal_id:     internal_id,
			player_id:       player.player.Player_id,
			turn_stamina:    10,
			cs:              createRisqCombatStats(),
			order_queue:     createRisqOrderQueue(),
			intent:          createRisqIntent(),
			target_priority: []TargetCategory{},
		},
		unit_id:     unit_id,
		stance:      UnitStance_PASSIVE,
		attack_back: true,
	}
	config, ok := unitConfigs[unit_id]
	if !ok {
		fmt.Fprintln(os.Stderr, "Creating unknown unit id: ", unit_id)
		return &unit
	}
	if config.unit_type != UnitType_ECONOMIC {
		unit.stance = UnitStance_DEFENSIVE
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
	unit.attack_range = config.attack_range
	unit.turn_stamina = config.turn_stamina
	for _, bonus := range bonusConfigs {
		if bonus.appliesTo(unit_id, config.unit_type) {
			applyTechBonus(&unit, bonus)
		}
	}
	for tech_id, researched := range player.researched_techs {
		if !researched {
			continue
		}
		tech, ok := techConfigs[tech_id]
		if !ok || !tech.appliesTo(unit_id, config.unit_type) {
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

func (u *RisqUnit) unitType() UnitType {
	return unitConfigs[u.unit_id].unit_type
}

func (u *RisqUnit) OrderableType() OrderableType {
	return OrderableType_UNIT
}

func (u *RisqUnit) combatStats(r *GameRisq, other Orderable, attacking bool) RisqCombatStats {
	return r.effectiveCombatStats(u, other, attacking)
}

func (u *RisqUnit) resolveHealthDelta(r *GameRisq) {
	if u.cs.pending_health_delta == 0 {
		return
	}
	was_alive := u.isAlive()
	u.cs.addHealth(u.cs.pending_health_delta)
	u.cs.pending_health_delta = 0
	if was_alive && !u.isAlive() {
		if event, ok := lowestIdAttackerEvent(u.attacked_by, r.current_tick); ok {
			if attacker := r.resolveAttacker(event); attacker != nil {
				u.recordDeath(r, attacker, event.damage)
			}
		}
	}
}

func (u *RisqUnit) recordDeath(r *GameRisq, attacker Attackable, damage float64) {
	space := u.zone.space.coordinate
	zone := u.zone.coordinate
	r.players[attacker.playerId()].report.recordCombat(RisqCombatEvent{tick: r.current_tick, kind: RisqCombatEventKind_UNIT_KILLED,
		self_player: attacker.playerId(), other_player: u.player_id, target_id: uint64(u.unit_id), space: space, zone: zone, damage: damage})
	r.players[u.player_id].report.recordCombat(RisqCombatEvent{tick: r.current_tick, kind: RisqCombatEventKind_UNIT_LOST,
		self_player: u.player_id, other_player: attacker.playerId(), target_id: uint64(u.unit_id), space: space, zone: zone, damage: damage})
	r.players[attacker.playerId()].kills++
	r.players[u.player_id].units_lost++
	u.deleted = true
}

func (u *RisqUnit) cleanupDeleted(risq *GameRisq) {
	resolveOrdersOnDeath(risq, &u.order_queue, u.internal_id, OrderType_UnitDelete)
	if u.zone != nil && u.zone.space != nil {
		u.zone.space.removeUnit(u)
	}
	if u.garrisoned_in != nil {
		delete(u.garrisoned_in.garrisoned_units, u.internal_id)
		u.garrisoned_in = nil
	}
	delete(risq.players[u.player_id].units, u.internal_id)
	delete(risq.units, u.internal_id)
}

func (u *RisqUnit) receiveOrder(o *RisqOrder, risq *GameRisq) error {
	switch o.order_type {
	case OrderType_UnitBuild:
		building_id, _, zone := invertBuildKey(uint(o.target_id), risq)
		player := risq.players[u.player_id]
		if zone.building == nil && player.planned_foundations[zone.coordinate_key] == nil {
			cost, _ := buildingProductionCost(building_id)
			if !player.resources.canAfford(cost) {
				return errors.New("cannot afford building")
			}
			player.planned_foundations[zone.coordinate_key] = createRisqPlannedFoundation(building_id, player)
		}
		u.order_queue.receiveOrder(o)
	case OrderType_UnitRenew:
		target := risq.buildings[uint64(o.target_id)]
		if target.renewing == nil {
			cost := buildingConfigs[target.building_id].gather.renew_cost
			player := risq.players[u.player_id]
			if !player.resources.canAfford(cost) {
				return errors.New("cannot afford renew")
			}
			player.resources.spend(cost)
			target.renewing = &cost
		}
		u.order_queue.receiveOrder(o)
	default:
		u.order_queue.receiveOrder(o)
	}
	return nil
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
	case OrderType_UnitGather:
		_, zone := invertZoneKey(uint(o.target_id), risq)
		if u.unitType() != UnitType_ECONOMIC || zone == nil {
			return false
		}
		if zone.resource != nil {
			resource, ok := zone.resourceKnownTo(u.player_id)
			return ok && resource.resources_left > 0
		}
		if b := zone.building; b != nil {
			config := buildingConfigs[b.building_id]
			if !config.isGatherable() || b.player_id != u.player_id || b.underConstruction() || b.resources_left <= 0 {
				return false
			}
			for _, ao := range u.order_queue.active_orders {
				if ao.order_type == OrderType_UnitGather && ao.target_id == o.target_id {
					return true
				}
			}
			return b.gatheringUnitCount(risq) < config.gather.gather_capacity
		}
		return false
	case OrderType_UnitRepair:
		target := risq.buildings[uint64(o.target_id)]
		if u.unitType() != UnitType_ECONOMIC || target == nil || target.isDeleted() {
			return false
		}
		cache, ok := target.zone.buildingKnownTo(u.player_id)
		return ok && cache.player_id == u.player_id && !cache.under_construction && cache.cs.health < float64(cache.cs.max_health)
	case OrderType_UnitRenew:
		target := risq.buildings[uint64(o.target_id)]
		if u.unitType() != UnitType_ECONOMIC || target == nil || target.isDeleted() || target.player_id != u.player_id || target.underConstruction() {
			return false
		}
		config := buildingConfigs[target.building_id]
		return config.isGatherable() && target.resources_left < config.gather.starting_resources
	case OrderType_UnitGarrison:
		target := risq.buildings[uint64(o.target_id)]
		return u.garrisonTargetValid(risq, target) && uint16(len(target.garrisoned_units)) < target.garrison_capacity && u.garrisoned_in != target
	case OrderType_UnitUngarrison:
		return u.garrisoned_in != nil
	case OrderType_UnitAttackUnit, OrderType_UnitAutoAttackUnit:
		target := risq.units[uint64(o.target_id)]
		if target == nil || target.zone == nil || !canAttack(u.player_id, target.player_id) {
			return false
		}
		if target.zone.space.getVisibility(u.player_id) < VisibilityGood {
			return false
		}
		return o.order_type != OrderType_UnitAutoAttackUnit || u.stanceReachable(target.zone)
	case OrderType_UnitAttackBuilding, OrderType_UnitAutoAttackBuilding:
		target := risq.buildings[uint64(o.target_id)]
		if target == nil || !canAttack(u.player_id, target.player_id) {
			return false
		}
		if _, ok := target.zone.buildingKnownTo(u.player_id); !ok {
			return false
		}
		return o.order_type != OrderType_UnitAutoAttackBuilding || u.stanceReachable(target.zone)
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

func (u *RisqUnit) garrisonTargetValid(risq *GameRisq, target *RisqBuilding) bool {
	return target != nil && !target.isDeleted() && !target.underConstruction() && risq.canAssist(u.player_id, target)
}

// reachableStatus reports the shared shape behind most orderStatus branches: an order that isn't
// satisfied yet stays InProgress as long as its subject can still get there, else it's Cancelled.
func reachableStatus(reachable bool) OrderStatus {
	if !reachable {
		return OrderStatus_Cancelled
	}
	return OrderStatus_InProgress
}

func (u *RisqUnit) orderStatus(o *RisqOrder, risq *GameRisq) OrderStatus {
	switch o.order_type {
	case OrderType_UnitMoveSpace:
		space := invertSpaceKey(uint(o.target_id), risq)
		start, _ := u.pathStart()
		if start == nil || start.space != space {
			return reachableStatus(u.canReach(space.getCenterZone(), RisqRange_ZONE))
		}
	case OrderType_UnitMoveZone:
		_, zone := invertZoneKey(uint(o.target_id), risq)
		if u.zone != zone {
			return reachableStatus(u.canReach(zone, RisqRange_ZONE))
		}
	case OrderType_UnitGather:
		_, zone := invertZoneKey(uint(o.target_id), risq)
		reachable := u.zone == zone || u.canReach(zone, RisqRange_ZONE)
		if resource, ok := zone.resourceKnownTo(u.player_id); ok && resource.resources_left > 0 {
			return reachableStatus(reachable)
		}
		if b := zone.building; b != nil && buildingConfigs[b.building_id].isGatherable() && b.resources_left > 0 {
			return reachableStatus(reachable)
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
				return OrderStatus_Cancelled
			}
			if u.zone == zone && !zone.space.buildableBy(u.player_id) {
				return OrderStatus_Cancelled
			}
			if u.zone != zone && !u.canReach(zone, RisqRange_ZONE) {
				return OrderStatus_Cancelled
			}
			return OrderStatus_InProgress
		}
		if zone.building.player_id != u.player_id || zone.building.building_id != building_id {
			risq.players[u.player_id].cancelPlannedFoundation(zone)
			return OrderStatus_Cancelled
		}
		if zone.building.underConstruction() {
			return reachableStatus(u.zone == zone || u.canReach(zone, RisqRange_ZONE))
		}
	case OrderType_UnitRepair:
		target := risq.buildings[uint64(o.target_id)]
		if target == nil || target.isDeleted() {
			return OrderStatus_Cancelled
		}
		cache, known := target.zone.buildingKnownTo(u.player_id)
		if known && !cache.under_construction && cache.player_id == u.player_id && cache.cs.health < float64(cache.cs.max_health) {
			if _, cost, ok := repairHealAndCost(target, 1); !ok || risq.players[u.player_id].resources.affordFraction(cost) <= 0 {
				return OrderStatus_Cancelled
			}
			return reachableStatus(u.zone == target.zone || u.canReach(target.zone, RisqRange_ZONE))
		}
	case OrderType_UnitRenew:
		target := risq.buildings[uint64(o.target_id)]
		if target == nil || target.isDeleted() {
			return OrderStatus_Cancelled
		}
		if target.player_id == u.player_id && target.renewing != nil {
			return reachableStatus(u.zone == target.zone || u.canReach(target.zone, RisqRange_ZONE))
		}
	case OrderType_UnitDelete:
		if !u.deleted {
			return OrderStatus_InProgress
		}
	case OrderType_UnitGarrison:
		target := risq.buildings[uint64(o.target_id)]
		if !u.garrisonTargetValid(risq, target) {
			return OrderStatus_Cancelled
		}
		if u.garrisoned_in != target {
			if uint16(len(target.garrisoned_units)) >= target.garrison_capacity {
				return OrderStatus_Cancelled
			}
			return reachableStatus(u.zone == target.zone || u.canReach(target.zone, RisqRange_ZONE))
		}
	case OrderType_UnitUngarrison:
		if u.garrisoned_in != nil {
			return OrderStatus_InProgress
		}
	case OrderType_UnitAttackBuilding, OrderType_UnitAutoAttackBuilding:
		target := risq.buildings[uint64(o.target_id)]
		if target == nil || target.isDeleted() {
			break
		}
		if !canAttack(u.player_id, target.player_id) {
			return OrderStatus_Cancelled
		}
		if _, ok := target.zone.buildingKnownTo(u.player_id); !ok {
			return OrderStatus_Cancelled
		}
		if o.order_type == OrderType_UnitAutoAttackBuilding && !u.stanceReachable(target.zone) {
			return OrderStatus_Cancelled
		}
		return reachableStatus(u.inAttackRange(target.zone) || u.canReach(target.zone, u.attack_range))
	case OrderType_UnitAttackUnit, OrderType_UnitAutoAttackUnit:
		target := risq.units[uint64(o.target_id)]
		if target == nil || target.isDeleted() {
			break
		}
		if !canAttack(u.player_id, target.player_id) {
			return OrderStatus_Cancelled
		}
		if target.zone == nil || target.zone.space.getVisibility(u.player_id) < VisibilityGood {
			return OrderStatus_Cancelled
		}
		if o.order_type == OrderType_UnitAutoAttackUnit && !u.stanceReachable(target.zone) {
			return OrderStatus_Cancelled
		}
		return reachableStatus(u.inAttackRange(target.zone) || u.canReach(target.zone, u.attack_range))
	case OrderType_UnitAttackZone:
		_, zone := invertZoneKey(uint(o.target_id), risq)
		if !u.inAttackRange(zone) {
			return reachableStatus(u.canReach(zone, u.attack_range))
		}
		if zoneHasEnemy(zone, u.player_id) {
			return OrderStatus_InProgress
		}
	case OrderType_UnitAttackSpace:
		space := invertSpaceKey(uint(o.target_id), risq)
		space_range, _ := u.attack_range.spaceRadius()
		start, _ := u.pathStart()
		if start == nil || game_utils.AxialDistance(start.space.coordinate, space.coordinate) > space_range {
			return reachableStatus(u.canReach(space.getCenterZone(), u.attack_range))
		}
		if spaceHasEnemy(space, u.player_id) {
			return OrderStatus_InProgress
		}
	}
	return OrderStatus_Executed
}

const aggressiveSightRadius = 2

func (u *RisqUnit) stanceReachable(target_zone *RisqZone) bool {
	if target_zone == nil || u.zone == nil {
		return false
	}
	if u.inAttackRange(target_zone) {
		return true
	}
	switch u.stance {
	case UnitStance_AGGRESSIVE:
		return game_utils.AxialDistance(u.zone.space.coordinate, target_zone.space.coordinate) <= aggressiveSightRadius
	case UnitStance_DEFENSIVE:
		return target_zone.space == u.zone.space
	case UnitStance_PASSIVE, UnitStance_STAND_GROUND:
		return false
	default:
		return false
	}
}

func (u *RisqUnit) isAttacking() bool {
	if len(u.order_queue.active_orders) == 0 {
		return false
	}
	switch u.order_queue.active_orders[0].order_type {
	case OrderType_UnitAttackUnit, OrderType_UnitAutoAttackUnit, OrderType_UnitAttackBuilding, OrderType_UnitAutoAttackBuilding, OrderType_UnitAttackZone, OrderType_UnitAttackSpace:
		return true
	default:
		return false
	}
}

func attackBackDistance(u *RisqUnit, attacker_zone *RisqZone) int {
	if attacker_zone.space == u.zone.space {
		return zoneDistanceWithinSpace(u.zone, attacker_zone)
	}
	return int(game_utils.AxialDistance(u.zone.space.coordinate, attacker_zone.space.coordinate)) * 6
}

// Among attackers that hit this tick, applies target_priority then nearest/lowest-id as a tiebreak.
func (u *RisqUnit) reactiveAttackBackTarget(risq *GameRisq) Attackable {
	if !u.attack_back || len(u.attacked_by) == 0 {
		return nil
	}
	latest_tick := u.attacked_by[len(u.attacked_by)-1].tick
	best := newCategoryBest()
	for _, event := range u.attacked_by {
		if event.tick != latest_tick {
			continue
		}
		switch event.attacker_type {
		case OrderableType_UNIT:
			attacker := risq.units[event.attacker_id]
			if attacker == nil || attacker.deleted {
				continue
			}
			if attacker.garrisoned_in != nil {
				building := attacker.garrisoned_in
				if building.deleted || building.zone == nil || !u.stanceReachable(building.zone) {
					continue
				}
				best.considerBuilding(building, attackBackDistance(u, building.zone))
				continue
			}
			if attacker.zone == nil || !u.stanceReachable(attacker.zone) {
				continue
			}
			best.considerUnit(attacker, attackBackDistance(u, attacker.zone))
		case OrderableType_BUILDING:
			attacker := risq.buildings[event.attacker_id]
			if attacker == nil || attacker.deleted || attacker.zone == nil || !u.stanceReachable(attacker.zone) {
				continue
			}
			best.considerBuilding(attacker, attackBackDistance(u, attacker.zone))
		}
	}
	return best.pick(u.target_priority)
}

// Bypasses player.active_orders bookkeeping, same as gather-point orders.
func (u *RisqUnit) replaceOrder(risq *GameRisq, order_type OrderType, target_id int64) {
	if len(u.order_queue.active_orders) > 0 {
		current := u.order_queue.active_orders[0]
		if current.order_type == order_type && current.target_id == target_id {
			return
		}
	}
	order := createRisqOrder(risq.nextOrderInternalId(), order_type, u.player_id, map[uint64]Orderable{u.internal_id: u}, target_id, false)
	if !u.orderReceivable(order, risq) {
		return
	}
	for _, ao := range append([]*RisqOrder(nil), u.order_queue.active_orders...) {
		u.cancelOrder(ao, risq)
	}
	u.receiveOrder(order, risq)
}

func (u *RisqUnit) resolveStance(risq *GameRisq) {
	if u.zone == nil || u.cs.attack_type == AttackType_NONE {
		return
	}
	if target := u.reactiveAttackBackTarget(risq); target != nil {
		idle_enough := u.interrupt_current
		if !idle_enough {
			if u.stance == UnitStance_PASSIVE {
				idle_enough = len(u.order_queue.active_orders) == 0
			} else {
				idle_enough = !u.isAttacking()
			}
		}
		if idle_enough {
			order_type := attackOrderType(target, OrderType_UnitAutoAttackUnit, OrderType_UnitAutoAttackBuilding)
			u.replaceOrder(risq, order_type, int64(target.internalId()))
		}
		return
	}
	if u.stance == UnitStance_PASSIVE {
		return
	}
	if !u.interrupt_current && len(u.order_queue.active_orders) > 0 {
		return
	}
	switch u.stance {
	case UnitStance_AGGRESSIVE:
		if target := nearbyAttackTarget(u.zone, u.player_id, u.target_priority, u.inAttackRange, risq, aggressiveSightRadius); target != nil {
			order_type := attackOrderType(target, OrderType_UnitAutoAttackUnit, OrderType_UnitAutoAttackBuilding)
			u.replaceOrder(risq, order_type, int64(target.internalId()))
		}
	case UnitStance_DEFENSIVE:
		if spaceHasEnemy(u.zone.space, u.player_id) {
			u.replaceOrder(risq, OrderType_UnitAttackSpace, int64(u.zone.space.coordinate_key))
		}
	case UnitStance_STAND_GROUND:
		if space_range, ranged := u.attack_range.spaceRadius(); ranged {
			if target := nearbyInRangeTarget(u.zone, u.player_id, u.target_priority, u.inAttackRange, risq, space_range); target != nil {
				order_type := attackOrderType(target, OrderType_UnitAutoAttackUnit, OrderType_UnitAutoAttackBuilding)
				u.replaceOrder(risq, order_type, int64(target.internalId()))
			}
		} else if zoneHasEnemy(u.zone, u.player_id) {
			u.replaceOrder(risq, OrderType_UnitAttackZone, int64(u.zone.coordinate_key))
		}
	}
}

// moveOrAct sets a move intent toward target unless arrived is true, in which case it runs act
// instead. Shared shape behind most tickIntent branches: approach a target, then do something once there.
func (u *RisqUnit) moveOrAct(arrived bool, target *RisqZone, move_range RisqRange, act func()) {
	if !arrived {
		u.intent.setMove(u.findPath(target, move_range))
		return
	}
	act()
}

func (u *RisqUnit) tickIntent(risq *GameRisq) bool {
	u.intent.resetIntent()
	u.resolveStance(risq)
	order := u.order_queue.nextOrder(u, risq)
	if order == nil {
		return false
	}
	switch order.order_type {
	case OrderType_UnitMoveSpace:
		space := invertSpaceKey(uint(order.target_id), risq)
		u.intent.setMove(u.findPath(space.getCenterZone(), RisqRange_ZONE))
	case OrderType_UnitMoveZone:
		_, zone := invertZoneKey(uint(order.target_id), risq)
		u.intent.setMove(u.findPath(zone, RisqRange_ZONE))
	case OrderType_UnitGather:
		_, zone := invertZoneKey(uint(order.target_id), risq)
		u.moveOrAct(u.zone == zone, zone, RisqRange_ZONE, func() {
			if zone.resource != nil {
				u.intent.setGather(zone.resource)
			} else if zone.building != nil {
				u.intent.setGather(zone.building)
			}
		})
	case OrderType_UnitBuild:
		building_id, _, zone := invertBuildKey(uint(order.target_id), risq)
		u.moveOrAct(u.zone == zone, zone, RisqRange_ZONE, func() {
			u.intent.setBuild(zone.building, building_id, zone)
		})
	case OrderType_UnitDelete:
		u.intent.setDelete()
	case OrderType_UnitAttackBuilding, OrderType_UnitAutoAttackBuilding:
		target := risq.buildings[uint64(order.target_id)]
		u.moveOrAct(u.inAttackRange(target.zone), target.zone, u.attack_range, func() {
			u.intent.setUnitAttack(target)
		})
	case OrderType_UnitAttackUnit, OrderType_UnitAutoAttackUnit:
		target := risq.units[uint64(order.target_id)]
		u.moveOrAct(u.inAttackRange(target.zone), target.zone, u.attack_range, func() {
			u.intent.setUnitAttack(target)
		})
	case OrderType_UnitAttackZone:
		_, zone := invertZoneKey(uint(order.target_id), risq)
		u.moveOrAct(u.inAttackRange(zone), zone, u.attack_range, func() {
			if target := zoneAttackTarget(zone, u.player_id, u.target_priority); target != nil {
				u.intent.setUnitAttack(target)
			}
		})
	case OrderType_UnitAttackSpace:
		space := invertSpaceKey(uint(order.target_id), risq)
		space_range, _ := u.attack_range.spaceRadius()
		start, _ := u.pathStart()
		in_space_range := start != nil && game_utils.AxialDistance(start.space.coordinate, space.coordinate) <= space_range
		u.moveOrAct(in_space_range, space.getCenterZone(), u.attack_range, func() {
			if target := spaceAttackTarget(u, space); target != nil {
				target_zone := target.currentZone()
				u.moveOrAct(u.inAttackRange(target_zone), target_zone, u.attack_range, func() {
					u.intent.setUnitAttack(target)
				})
			}
		})
	case OrderType_UnitRepair:
		target := risq.buildings[uint64(order.target_id)]
		if target == nil {
			break
		}
		u.moveOrAct(u.zone == target.zone, target.zone, RisqRange_ZONE, func() {
			u.intent.setRepair(target)
		})
	case OrderType_UnitRenew:
		target := risq.buildings[uint64(order.target_id)]
		if target == nil {
			break
		}
		u.moveOrAct(u.zone == target.zone, target.zone, RisqRange_ZONE, func() {
			u.intent.setRenew(target)
		})
	case OrderType_UnitGarrison:
		target := risq.buildings[uint64(order.target_id)]
		if target == nil {
			break
		}
		if u.garrisoned_in == target {
			break
		}
		u.moveOrAct(u.zone == target.zone, target.zone, RisqRange_ZONE, func() {
			u.intent.setGarrison(target)
		})
	case OrderType_UnitUngarrison:
		if u.garrisoned_in == nil {
			break
		}
		u.intent.setUngarrison(u.garrisoned_in.zone)
	default:
		fmt.Fprintln(os.Stderr, "Order type not implemented:", order.order_type)
	}

	// Automatic ungarrison if we have an intent that requires being on the map
	if u.garrisoned_in != nil && u.intent.hasIntent() {
		switch u.intent.detail.(type) {
		case *GarrisonIntent, *UngarrisonIntent:
		default:
			u.intent.setUngarrison(u.garrisoned_in.zone)
		}
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
		util.DebugLog.Println("Moving unit", u.display_name, "to zone"+detail.next_step.coordinate.ToString(), "in space", detail.next_step.space.coordinate.ToString())
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
			amount = float64(u.intent.intent_cost) * (float64(detail.source.gatherSpeed()) / gatherRateStaminaBase)
		}
		amount = util.RoundTo(amount, 4)
		if amount > detail.source.gatherResourcesLeft() {
			amount = detail.source.gatherResourcesLeft()
		}
		detail.source.gatherDrain(amount)
		risq.players[u.player_id].resources.addGathered(detail.source.gatherCategory(), amount)
	case *ConstructionIntent:
		if detail.zone.building != nil && detail.zone.building.player_id != u.player_id {
			return
		}
		building := detail.building_under_construction
		if building == nil {
			if detail.zone.building != nil {
				building = detail.zone.building
			} else {
				if !detail.zone.space.buildableBy(u.player_id) {
					return
				}
				if winner, ok := risq.construction_winners[detail.zone]; ok && winner != u.internal_id {
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
				risq.cancelPendingTerrainClear(detail.zone)
				detail.zone.terrain_override = 0
			}
		}
		if !building.deleted && building.underConstruction() {
			old_ratio := constructionHealthRatio(building.stamina_remaining, building.construction_stamina_total)
			progress := max(1, int(math.Round(float64(u.intent.intent_cost)*building.zone.space.buildSpeedModifier())))
			building.stamina_remaining -= progress
			new_ratio := constructionHealthRatio(building.stamina_remaining, building.construction_stamina_total)
			building.cs.queueHealth(float64(building.cs.max_health) * (new_ratio - old_ratio))
			if !building.underConstruction() {
				risq.players[building.player_id].report.recordBuildingBuilt(building.building_id, building.zone.space.coordinate, building.zone.coordinate)
				building.refreshTerrainOverride()
				if buildingConfigs[building.building_id].isGatherable() {
					risq.completed_gatherables = append(risq.completed_gatherables, building)
				}
			}
		}
	case *DeleteIntent:
		u.deleted = true
		risq.players[u.player_id].units_lost++
		// TODO: check for recent damage to assign kill
	case *UnitAttackIntent:
		risq.unitAttack(u, detail.target)
	case *RepairIntent:
		building := detail.target
		if building.deleted || building.underConstruction() || !risq.canAssist(u.player_id, building) {
			return
		}
		heal, cost, ok := repairHealAndCost(building, u.intent.intent_cost)
		if !ok {
			return
		}
		afford := risq.repair_allotments[u]
		if afford <= 0 {
			return
		}
		building.cs.queueHealth(heal * afford)
		risq.players[u.player_id].resources.spend(cost.scale(afford))
	case *RenewIntent:
		building := detail.target
		if building.deleted || building.renewing == nil {
			return
		}
		config := buildingConfigs[building.building_id]
		if config.gather.renew_stamina <= 0 {
			return
		}
		delta := config.gather.starting_resources * float64(u.intent.intent_cost) / float64(config.gather.renew_stamina)
		building.resources_left = min(building.resources_left+delta, config.gather.starting_resources)
		if building.resources_left >= config.gather.starting_resources {
			building.renewing = nil
		}
		building.refreshTerrainOverride()
	case *GarrisonIntent:
		target := detail.target
		if !u.deleted && u.garrisonTargetValid(risq, target) && u.zone == target.zone && risq.garrison_allotments[u] {
			target.garrisoned_units[u.internal_id] = u
			u.garrisoned_in = target
			u.zone.space.removeUnit(u)
			u.zone = nil
		}
	case *UngarrisonIntent:
		if u.garrisoned_in == nil {
			return
		}
		building := u.garrisoned_in
		delete(building.garrisoned_units, u.internal_id)
		u.garrisoned_in = nil
		building.zone.space.setUnit(&building.zone.coordinate, u)
	}
	u.current_stamina -= u.intent.intent_cost
}

func (u *RisqUnit) toFrontend(viewer_player_id int) gin.H {
	unit := gin.H{
		"internal_id":     u.internal_id,
		"player_id":       u.player_id,
		"unit_id":         u.unit_id,
		"unit_type":       u.unitType(),
		"display_name":    u.display_name,
		"turn_stamina":    u.turn_stamina,
		"current_stamina": u.current_stamina,
		"max_stamina":     maxStaminaFor(u.turn_stamina),
		"combat_stats":    u.cs.toFrontend(),
		"attack_range":    u.attack_range,
	}
	if u.garrisoned_in != nil {
		unit["garrisoned_in"] = u.garrisoned_in.internal_id
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
		target_priority := make([]int, len(u.target_priority))
		for i, cat := range u.target_priority {
			target_priority[i] = int(cat)
		}
		unit["stance"] = u.stance
		unit["interrupt_current"] = u.interrupt_current
		unit["attack_back"] = u.attack_back
		unit["target_priority"] = target_priority
	}
	unit["active_orders"] = active_orders
	return unit
}
