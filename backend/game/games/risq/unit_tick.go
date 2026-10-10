package risq

import (
	"fmt"
	"math"
	"os"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
)

// moveOrAct sets a move intent toward target unless arrived is true, in which case it runs act instead
func (u *RisqUnit) moveOrAct(arrived bool, target *RisqZone, move_range defs.RisqRange, act func()) {
	if !arrived {
		u.intent.setMove(u.findPath(target, move_range))
		return
	}
	act()
}

// Records the unit a move is closing on, so a melee approach can meet a unit crossing the same border
func (u *RisqUnit) markChase(target Attackable) {
	if move, ok := u.intent.detail.(*MoveIntent); ok {
		move.target = target
		move.chasing, _ = target.(*RisqUnit)
	}
}

func (u *RisqUnit) tickIntent(risq *GameRisq) bool {
	u.intent.resetIntent()
	paid_half_move := u.half_move
	u.half_move = nil
	if len(u.order_queue.active_orders) > 0 {
		order_type := u.order_queue.active_orders[0].order_type
		if order_type == defs.OrderType_UnitAutoAttackUnit || order_type == defs.OrderType_UnitAutoAttackBuilding {
			u.order_queue.nextOrder(u, risq)
		}
	}
	u.resolveStance(risq)
	order := u.order_queue.nextOrder(u, risq)
	if order == nil {
		return false
	}
	switch order.order_type {
	case defs.OrderType_UnitMoveSpace:
		space := invertSpaceKey(uint(order.target_id), risq)
		u.intent.setMove(u.findPath(space.getCenterZone(), defs.RisqRange_ZONE))
	case defs.OrderType_UnitMoveZone:
		_, zone := invertZoneKey(uint(order.target_id), risq)
		u.intent.setMove(u.findPath(zone, defs.RisqRange_ZONE))
	case defs.OrderType_UnitGather:
		_, zone := invertZoneKey(uint(order.target_id), risq)
		u.moveOrAct(u.zone == zone, zone, defs.RisqRange_ZONE, func() {
			if zone.resource != nil {
				u.intent.setGather(zone.resource)
			} else if zone.building != nil {
				u.intent.setGather(zone.building)
			}
		})
	case defs.OrderType_UnitBuild:
		building_id, _, zone := invertBuildKey(uint(order.target_id), risq)
		u.moveOrAct(u.zone == zone, zone, defs.RisqRange_ZONE, func() {
			u.intent.setBuild(zone.building, building_id, zone)
		})
	case defs.OrderType_UnitDelete:
		u.intent.setDelete()
	case defs.OrderType_UnitAttackBuilding, defs.OrderType_UnitAutoAttackBuilding:
		target := risq.buildings[uint64(order.target_id)]
		u.moveOrAct(u.inAttackRange(target.zone), target.zone, u.attack_range, func() {
			u.intent.setUnitAttack(target)
		})
		u.markChase(target)
	case defs.OrderType_UnitAttackUnit, defs.OrderType_UnitAutoAttackUnit:
		target := risq.units[uint64(order.target_id)]
		u.moveOrAct(u.inAttackRange(target.zone), target.zone, u.attack_range, func() {
			u.intent.setUnitAttack(target)
		})
		u.markChase(target)
	case defs.OrderType_UnitAttackZone:
		_, zone := invertZoneKey(uint(order.target_id), risq)
		u.moveOrAct(u.inAttackRange(zone), zone, u.attack_range, func() {
			if target := zoneAttackTarget(zone, u.player_id, u.target_priority); target != nil {
				u.intent.setUnitAttack(target)
			}
		})
		if u.attack_range == defs.RisqRange_SPACE {
			u.markChase(zoneAttackTarget(zone, u.player_id, u.target_priority))
		}
	case defs.OrderType_UnitAttackSpace:
		space := invertSpaceKey(uint(order.target_id), risq)
		if u.garrisoned_in != nil {
			u.intent.setUngarrison(u.garrisoned_in)
			break
		}
		space_range, _ := u.attack_range.SpaceRadius()
		start, _ := u.pathStart()
		in_space_range := start != nil && start.space.distanceTo(space) <= space_range
		u.moveOrAct(in_space_range, space.getCenterZone(), u.attack_range, func() {
			if target := spaceAttackTarget(u, space); target != nil {
				target_zone := target.currentZone()
				u.moveOrAct(u.inAttackRange(target_zone), target_zone, u.attack_range, func() {
					u.intent.setUnitAttack(target)
				})
				u.markChase(target)
			}
		})
		if move, ok := u.intent.detail.(*MoveIntent); ok && u.attack_range == defs.RisqRange_SPACE && move.next_step.space == space {
			u.markChase(zoneAttackTarget(move.next_step, u.player_id, u.target_priority))
		}
	case defs.OrderType_UnitRepair:
		target := risq.buildings[uint64(order.target_id)]
		if target == nil {
			break
		}
		u.moveOrAct(u.zone == target.zone, target.zone, defs.RisqRange_ZONE, func() {
			u.intent.setRepair(target)
		})
	case defs.OrderType_UnitRenew:
		target := risq.buildings[uint64(order.target_id)]
		if target == nil {
			break
		}
		u.moveOrAct(u.zone == target.zone, target.zone, defs.RisqRange_ZONE, func() {
			u.intent.setRenew(target)
		})
	case defs.OrderType_UnitGarrison:
		target := risq.buildings[uint64(order.target_id)]
		if target == nil {
			break
		}
		if u.garrisoned_in == target {
			break
		}
		u.moveOrAct(u.zone == target.zone, target.zone, defs.RisqRange_ZONE, func() {
			u.intent.setGarrison(target)
		})
	case defs.OrderType_UnitUngarrison:
		if u.garrisoned_in == nil {
			break
		}
		u.intent.setUngarrison(u.garrisoned_in)
	default:
		fmt.Fprintln(os.Stderr, "Order type not implemented:", order.order_type)
	}

	// Automatic ungarrison if we have an intent that requires being on the map
	if u.garrisoned_in != nil && u.intent.hasIntent() {
		risq.recordTickIntent(u, order)
		switch u.intent.detail.(type) {
		case *GarrisonIntent, *UngarrisonIntent:
		default:
			u.intent.setUngarrison(u.garrisoned_in)
		}
	}

	if move, ok := u.intent.detail.(*MoveIntent); ok {
		u.move_path = move.path
	} else {
		u.move_path = nil
	}

	u.restoreHalfMove(paid_half_move)
	if u.tick_action == nil {
		risq.recordTickIntent(u, order)
	}
	u.intent.resolveCost(u.current_stamina)
	if !u.intent.hasIntent() {
		u.half_move = nil
	}
	return u.intent.hasIntent()
}

func (u *RisqUnit) tickExecute(risq *GameRisq) {
	if !u.intent.hasIntent() {
		return
	}
	stamina_before := u.current_stamina
	u.startTickExecution()
	defer u.finishTickAction(stamina_before)
	if _, gathering := u.intent.detail.(*GatherIntent); !gathering {
		u.gather_slot = nil
	}
	switch detail := u.intent.detail.(type) {
	case *MoveIntent:
		util.DebugLog.Printf("Moving unit %d %s to zone%s in space %s tick=%d turn=%d stamina=%d from=%s",
			u.internal_id, u.display_name, detail.next_step.coordinate.ToString(), detail.next_step.space.coordinate.ToString(), risq.current_tick, risq.turn_number, u.intent.intent_cost, u.zone.space.coordinate.ToString())
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
		if amount == gatherDenied {
			u.blockTickAction("gather_capacity")
			u.gather_slot = nil
			return
		}
		u.gather_slot = detail.source
		if !ok {
			amount = float64(u.intent.intent_cost) * (float64(detail.source.gatherSpeed()) / gatherRateStaminaBase)
		}
		amount = util.RoundTo(amount, 4)
		if amount > detail.source.gatherResourcesLeft() {
			amount = detail.source.gatherResourcesLeft()
		}
		if u.tick_action != nil {
			u.tick_action.execute.gathered = amount
		}
		detail.source.gatherDrain(amount)
		risq.players[u.player_id].resources.addGathered(detail.source.gatherCategory(), amount)
	case *ConstructionIntent:
		if detail.zone.building != nil && detail.zone.building.player_id != u.player_id {
			u.blockTickAction("foundation_lost")
			return
		}
		building := detail.building_under_construction
		if building == nil {
			if winner, founding := risq.construction_winners[detail.zone]; founding && winner != u.internal_id {
				u.blockTickAction("foundation_lost")
				return
			}
			if detail.zone.building != nil {
				building = detail.zone.building
			} else {
				foundation_id, buildable := risq.foundation_ids[detail.zone]
				if !buildable {
					u.blockTickAction("target_invalid")
					return
				}
				_, stamina_required := defs.BuildingProductionCost(detail.building_id)
				building = createRisqBuilding(foundation_id, detail.building_id, u.player_id)
				util.DebugLog.Printf("Foundation started: building=%d building_id=%d player=%d builder=%d zone=%s space=%s tick=%d",
					building.internal_id, detail.building_id, u.player_id, u.internal_id, detail.zone.coordinate.ToString(), detail.zone.space.coordinate.ToString(), risq.current_tick)
				building.stamina_remaining = stamina_required
				building.construction_stamina_total = stamina_required
				building.health_synced_stamina = stamina_required
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
			progress := max(1, int(math.Round(float64(u.intent.intent_cost)*building.zone.space.buildSpeedModifier())))
			u.recordTickProgress(min(building.stamina_remaining, progress))
			if u.tick_action != nil {
				location := tickZoneLocation(building.zone)
				u.tick_action.execute.target, u.tick_action.execute.target_location = tickActorTarget(building), &location
				u.tick_action.execute_target_visibility = tickZoneVisibility(building.zone)
			}
			building.stamina_remaining = max(0, building.stamina_remaining-progress)
			if !building.underConstruction() {
				util.DebugLog.Printf("Construction complete: building=%d building_id=%d player=%d zone=%s space=%s tick=%d",
					building.internal_id, building.building_id, building.player_id, building.zone.coordinate.ToString(), building.zone.space.coordinate.ToString(), risq.current_tick)
				risq.players[building.player_id].report.recordBuildingBuilt(building.building_id, building.zone.space.coordinate, building.zone.coordinate)
				building.refreshTerrainOverride()
				if defs.BuildingConfigs[building.building_id].IsGatherable() {
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
			u.blockTickAction("target_invalid")
			return
		}
		afford := risq.repair_allotments[u]
		if afford <= 0 {
			u.blockTickAction("cannot_afford_repair")
			return
		}
	case *RenewIntent:
		building := detail.target
		if building.deleted || building.renewing == nil {
			u.blockTickAction("target_invalid")
			return
		}
		building.pending_renew_stamina += u.intent.intent_cost
		u.recordTickProgress(u.intent.intent_cost)
	case *GarrisonIntent:
		target := detail.target
		if !u.deleted && u.garrisonTargetValid(risq, target) && u.zone == target.zone && risq.garrison_allotments[u] {
			target.garrisoned_units[u.internal_id] = u
			u.garrisoned_in = target
			u.zone.space.removeUnit(u)
			u.zone = nil
		} else {
			u.blockTickAction("garrison_full")
		}
	case *UngarrisonIntent:
		if u.garrisoned_in == nil {
			u.blockTickAction("target_invalid")
			return
		}
		building := u.garrisoned_in
		delete(building.garrisoned_units, u.internal_id)
		u.garrisoned_in = nil
		building.zone.space.setUnit(&building.zone.coordinate, u)
	}
	u.current_stamina -= u.intent.intent_cost
}
