package risq

import (
	"errors"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

func (u *RisqUnit) receiveOrder(o *RisqOrder, risq *GameRisq, prepend bool) error {
	switch o.order_type {
	case defs.OrderType_UnitBuild:
		building_id, _, zone := invertBuildKey(uint(o.target_id), risq)
		player := risq.players[u.player_id]
		if zone.building == nil && player.planned_foundations[zone.coordinate_key] == nil {
			cost, _ := defs.BuildingProductionCost(building_id)
			if !player.resources.canAfford(cost) {
				return errors.New("cannot afford building")
			}
			player.planned_foundations[zone.coordinate_key] = createRisqPlannedFoundation(building_id, player)
		}
		u.order_queue.receiveOrder(o, prepend)
	case defs.OrderType_UnitRenew:
		target := risq.buildings[uint64(o.target_id)]
		if target.renewing == nil {
			cost := defs.BuildingConfigs[target.building_id].Gather.Renew_cost
			player := risq.players[u.player_id]
			if !player.resources.canAfford(cost) {
				return errors.New("cannot afford renew")
			}
			player.resources.spend(cost)
			target.renewing = &cost
		}
		u.order_queue.receiveOrder(o, prepend)
	default:
		u.order_queue.receiveOrder(o, prepend)
	}
	return nil
}

func (u *RisqUnit) cancelOrder(o *RisqOrder, risq *GameRisq) {
	u.order_queue.removeOrder(o.internal_id)
	o.resolveSubject(u.internal_id, false, risq.turn_number)
	switch o.order_type {
	case defs.OrderType_UnitBuild:
		// the planned foundation is independent of the order that created it; cancelling the order doesn't touch it
	}
}

func (u *RisqUnit) orderReceivable(o *RisqOrder, risq *GameRisq) bool {
	switch o.order_type {
	case defs.OrderType_UnitBuild:
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
	case defs.OrderType_UnitGather:
		_, zone := invertZoneKey(uint(o.target_id), risq)
		if u.unitType() != defs.UnitType_ECONOMIC || zone == nil {
			return false
		}
		// judged on what the player knows: a resource it last saw in fog stays a valid target until it looks again
		if resource, ok := zone.resourceKnownTo(u.player_id); ok {
			return resource.resources_left > 0
		}
		if b := zone.building; b != nil {
			config := defs.BuildingConfigs[b.building_id]
			return config.IsGatherable() && !b.deleted && b.player_id == u.player_id && !b.underConstruction() && b.resources_left > 0
		}
		return false
	case defs.OrderType_UnitRepair:
		target := risq.buildings[uint64(o.target_id)]
		if u.unitType() != defs.UnitType_ECONOMIC || target == nil || target.isDeleted() {
			return false
		}
		cache, ok := target.zone.buildingKnownTo(u.player_id)
		return ok && cache.player_id == u.player_id && !cache.under_construction && cache.cs.health < float64(cache.cs.max_health)
	case defs.OrderType_UnitRenew:
		target := risq.buildings[uint64(o.target_id)]
		if u.unitType() != defs.UnitType_ECONOMIC || target == nil || target.isDeleted() || target.player_id != u.player_id || target.underConstruction() {
			return false
		}
		config := defs.BuildingConfigs[target.building_id]
		return config.IsGatherable() && target.resources_left <= 0
	case defs.OrderType_UnitGarrison:
		target := risq.buildings[uint64(o.target_id)]
		return u.garrisonTargetValid(risq, target) && uint16(len(target.garrisoned_units)) < target.garrison_capacity && u.garrisoned_in != target
	case defs.OrderType_UnitUngarrison:
		return u.garrisoned_in != nil
	case defs.OrderType_UnitAttackUnit, defs.OrderType_UnitAutoAttackUnit:
		target := risq.units[uint64(o.target_id)]
		if target == nil || target.zone == nil || !canAttack(u.player_id, target.player_id) {
			return false
		}
		if target.zone.space.getVisibility(u.player_id) < defs.VisibilityGood {
			return false
		}
		return o.order_type != defs.OrderType_UnitAutoAttackUnit || u.stanceReachable(target.zone)
	case defs.OrderType_UnitAttackBuilding, defs.OrderType_UnitAutoAttackBuilding:
		target := risq.buildings[uint64(o.target_id)]
		if target == nil || !canAttack(u.player_id, target.player_id) {
			return false
		}
		if _, ok := target.zone.buildingKnownTo(u.player_id); !ok {
			return false
		}
		return o.order_type != defs.OrderType_UnitAutoAttackBuilding || u.stanceReachable(target.zone)
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
	case defs.OrderType_UnitMoveSpace:
		space := invertSpaceKey(uint(o.target_id), risq)
		start, _ := u.pathStart()
		if start == nil || start.space != space {
			return reachableStatus(u.canReach(space.getCenterZone(), defs.RisqRange_ZONE))
		}
	case defs.OrderType_UnitMoveZone:
		_, zone := invertZoneKey(uint(o.target_id), risq)
		if u.zone != zone {
			return reachableStatus(u.canReach(zone, defs.RisqRange_ZONE))
		}
	case defs.OrderType_UnitGather:
		_, zone := invertZoneKey(uint(o.target_id), risq)
		if u.gatherSlotsVisiblyFull(zone, risq) {
			return OrderStatus_Cancelled
		}
		reachable := u.zone == zone || u.canReach(zone, defs.RisqRange_ZONE)
		if resource, ok := zone.resourceKnownTo(u.player_id); ok && resource.resources_left > 0 {
			return reachableStatus(reachable)
		}
		if b := zone.building; b != nil && !b.deleted && b.player_id == u.player_id && !b.underConstruction() && defs.BuildingConfigs[b.building_id].IsGatherable() && b.resources_left > 0 {
			return reachableStatus(reachable)
		}
	case defs.OrderType_UnitBuild:
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
			if u.zone != zone && !u.canReach(zone, defs.RisqRange_ZONE) {
				return OrderStatus_Cancelled
			}
			return OrderStatus_InProgress
		}
		if zone.building.player_id != u.player_id || zone.building.building_id != building_id {
			risq.players[u.player_id].cancelPlannedFoundation(zone)
			return OrderStatus_Cancelled
		}
		if zone.building.underConstruction() {
			return reachableStatus(u.zone == zone || u.canReach(zone, defs.RisqRange_ZONE))
		}
	case defs.OrderType_UnitRepair:
		target := risq.buildings[uint64(o.target_id)]
		if target == nil || target.isDeleted() {
			return OrderStatus_Cancelled
		}
		cache, known := target.zone.buildingKnownTo(u.player_id)
		if known && !cache.under_construction && cache.player_id == u.player_id && cache.cs.health < float64(cache.cs.max_health) {
			if _, cost, ok := repairHealAndCost(target, 1); !ok || risq.players[u.player_id].resources.affordFraction(cost) <= 0 {
				return OrderStatus_Cancelled
			}
			return reachableStatus(u.zone == target.zone || u.canReach(target.zone, defs.RisqRange_ZONE))
		}
	case defs.OrderType_UnitRenew:
		target := risq.buildings[uint64(o.target_id)]
		if target == nil || target.isDeleted() {
			return OrderStatus_Cancelled
		}
		if target.player_id == u.player_id && target.renewing != nil {
			return reachableStatus(u.zone == target.zone || u.canReach(target.zone, defs.RisqRange_ZONE))
		}
	case defs.OrderType_UnitDelete:
		if !u.deleted {
			return OrderStatus_InProgress
		}
	case defs.OrderType_UnitGarrison:
		target := risq.buildings[uint64(o.target_id)]
		if !u.garrisonTargetValid(risq, target) {
			return OrderStatus_Cancelled
		}
		if u.garrisoned_in != target {
			if uint16(len(target.garrisoned_units)) >= target.garrison_capacity {
				return OrderStatus_Cancelled
			}
			return reachableStatus(u.zone == target.zone || u.canReach(target.zone, defs.RisqRange_ZONE))
		}
	case defs.OrderType_UnitUngarrison:
		if u.garrisoned_in != nil {
			return OrderStatus_InProgress
		}
	case defs.OrderType_UnitAttackBuilding, defs.OrderType_UnitAutoAttackBuilding:
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
		if o.order_type == defs.OrderType_UnitAutoAttackBuilding && !u.stanceReachable(target.zone) {
			return OrderStatus_Cancelled
		}
		return reachableStatus(u.inAttackRange(target.zone) || u.canReach(target.zone, u.attack_range))
	case defs.OrderType_UnitAttackUnit, defs.OrderType_UnitAutoAttackUnit:
		target := risq.units[uint64(o.target_id)]
		if target == nil || target.isDeleted() {
			break
		}
		if !canAttack(u.player_id, target.player_id) {
			return OrderStatus_Cancelled
		}
		if target.zone == nil || target.zone.space.getVisibility(u.player_id) < defs.VisibilityGood {
			return OrderStatus_Cancelled
		}
		if o.order_type == defs.OrderType_UnitAutoAttackUnit && !u.stanceReachable(target.zone) {
			return OrderStatus_Cancelled
		}
		return reachableStatus(u.inAttackRange(target.zone) || u.canReach(target.zone, u.attack_range))
	case defs.OrderType_UnitAttackZone:
		_, zone := invertZoneKey(uint(o.target_id), risq)
		if !u.inAttackRange(zone) {
			return reachableStatus(u.canReach(zone, u.attack_range))
		}
		if zoneHasEnemy(zone, u.player_id) {
			return OrderStatus_InProgress
		}
	case defs.OrderType_UnitAttackSpace:
		space := invertSpaceKey(uint(o.target_id), risq)
		space_range, _ := u.attack_range.SpaceRadius()
		start, _ := u.pathStart()
		if start == nil || start.space.distanceTo(space) > space_range {
			return reachableStatus(u.canReach(space.getCenterZone(), u.attack_range))
		}
		if spaceHasEnemy(space, u.player_id) {
			return OrderStatus_InProgress
		}
	}
	return OrderStatus_Executed
}

// True when u doesn't hold a slot at the zone's source and the holders its player can see already fill it
func (u *RisqUnit) gatherSlotsVisiblyFull(zone *RisqZone, risq *GameRisq) bool {
	source := zoneGatherSource(zone)
	if source == nil || u.gather_slot == source {
		return false
	}
	return gatherSlotHolders(source, zone, risq, u.player_id, u) >= source.gatherCapacity()
}
