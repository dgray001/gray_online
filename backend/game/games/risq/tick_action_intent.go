package risq

import "github.com/dgray001/gray_online/game/games/risq/internal/defs"

func (r *GameRisq) newTickAction(actor Orderable, order *RisqOrder) *TickAction {
	if r.tick_history == nil || !r.tick_history.recording {
		return nil
	}
	base := tickBase(actor)
	action := &TickAction{
		tick: r.tick_history.tick, player_id: base.player_id,
		order: tickOrderValue(order), visibility: tickZoneVisibility(tickActorZone(actor)),
		intent:  tickIntent{location: tickActorLocation(actor), available_stamina: base.current_stamina},
		execute: tickExecution{kind: "none", outcome: "skipped", effects: []tickEffect{}},
	}
	if n := len(base.tick_actions); n > 0 && base.tick_actions[n-1].tick == action.tick {
		action.sequence = base.tick_actions[n-1].sequence + 1
	}
	base.tick_actions = append(base.tick_actions, action)
	target := tickActorTarget(actor)
	r.tick_history.histories[tickActorKey{target.kind, target.internal_id}] = base.tick_actions
	return action
}

func (a *TickAction) targetSnapshot(target tickTarget, zone *RisqZone) {
	location := tickZoneLocation(zone)
	a.intent.target, a.intent.target_location = target, &location
	a.target_visibility = tickZoneVisibility(zone)
}

func (r *GameRisq) recordTickIntent(actor Orderable, order *RisqOrder) {
	action := r.newTickAction(actor, order)
	if action == nil {
		return
	}
	base := tickBase(actor)
	base.tick_action = action
	action.intent.kind = tickOrderKind(order.order_type)
	action.intent.min_cost, action.intent.max_cost = base.intent.min_cost, base.intent.max_cost
	r.snapshotIntent(action, base.intent)
}

func (r *GameRisq) snapshotIntent(action *TickAction, intent *RisqIntent) {
	switch detail := intent.detail.(type) {
	case *MoveIntent:
		action.intent.kind = "move"
		location := tickZoneLocation(detail.next_step)
		action.intent.destination = &location
		if detail.target != nil {
			action.targetSnapshot(tickActorTarget(detail.target), tickActorZone(detail.target))
		}
	case *GatherIntent:
		action.intent.kind = "gather"
		switch source := detail.source.(type) {
		case *RisqResource:
			action.targetSnapshot(tickTarget{"resource", source.internal_id, -1}, source.zone)
		case *RisqBuilding:
			action.targetSnapshot(tickActorTarget(source), source.zone)
		}
	case *UnitAttackIntent:
		action.intent.kind = "attack"
		action.targetSnapshot(tickActorTarget(detail.target), tickActorZone(detail.target))
	case *BuildingAttackIntent:
		action.intent.kind = "attack"
		action.targetSnapshot(tickActorTarget(detail.target), tickActorZone(detail.target))
	case *ConstructionIntent:
		action.intent.kind = "build"
		action.intent.item_id = detail.building_id
		location := tickZoneLocation(detail.zone)
		action.intent.destination = &location
		if detail.zone.building != nil {
			action.targetSnapshot(tickActorTarget(detail.zone.building), detail.zone)
		}
	case *RepairIntent:
		action.intent.kind = "repair"
		action.targetSnapshot(tickActorTarget(detail.target), detail.target.zone)
	case *RenewIntent:
		action.intent.kind = "renew"
		action.targetSnapshot(tickActorTarget(detail.target), detail.target.zone)
	case *GarrisonIntent:
		action.intent.kind = "garrison"
		action.targetSnapshot(tickActorTarget(detail.target), detail.target.zone)
	case *UngarrisonIntent:
		action.intent.kind = "ungarrison"
		action.targetSnapshot(tickActorTarget(detail.building), detail.building.zone)
	case *ProductionIntent:
		action.intent.kind = "production"
		action.intent.item_id, action.intent.producible_kind = detail.item.item_id, detail.item.kind
	case *DeleteIntent:
		action.intent.kind = "delete"
	}
}

func tickOrderKind(kind defs.OrderType) string {
	switch kind {
	case defs.OrderType_UnitMoveSpace, defs.OrderType_UnitMoveZone:
		return "move"
	case defs.OrderType_UnitGather:
		return "gather"
	case defs.OrderType_UnitBuild:
		return "build"
	case defs.OrderType_UnitRepair:
		return "repair"
	case defs.OrderType_UnitRenew:
		return "renew"
	case defs.OrderType_UnitGarrison:
		return "garrison"
	case defs.OrderType_UnitUngarrison:
		return "ungarrison"
	case defs.OrderType_UnitDelete, defs.OrderType_BuildingDelete:
		return "delete"
	case defs.OrderType_BuildingCreate, defs.OrderType_BuildingResearch:
		return "production"
	default:
		return "attack"
	}
}

func (b *orderableBase) blockTickAction(reason string) {
	if b.tick_action != nil {
		b.tick_action.execute.outcome, b.tick_action.execute.reason = "blocked", reason
	}
}

func (b *orderableBase) finishTickAction(stamina_before int) {
	if b.tick_action != nil {
		b.tick_action.execute.stamina_spent = stamina_before - b.current_stamina
	}
}

func (r *GameRisq) recordOrderDecision(actor Orderable, order *RisqOrder, status OrderStatus) {
	action := r.newTickAction(actor, order)
	if action == nil {
		return
	}
	action.intent.kind = tickOrderKind(order.order_type)
	action.execute.outcome = "executed"
	action.execute.kind = "order_completed"
	if status == OrderStatus_Cancelled {
		action.execute.kind = "order_cancelled"
		action.execute.outcome, action.execute.reason = "blocked", r.tickCancellationReason(actor, order)
	} else {
		action.execute.reason = "order_completed"
	}
}

func (r *GameRisq) tickCancellationReason(actor Orderable, order *RisqOrder) string {
	switch order.order_type {
	case defs.OrderType_UnitMoveSpace, defs.OrderType_UnitMoveZone:
		return "no_reachable_target"
	case defs.OrderType_UnitGather:
		if unit, ok := actor.(*RisqUnit); ok {
			_, zone := invertZoneKey(uint(order.target_id), r)
			if unit.gatherSlotsVisiblyFull(zone, r) {
				return "gather_capacity"
			}
		}
	case defs.OrderType_UnitBuild:
		_, _, zone := invertBuildKey(uint(order.target_id), r)
		if zone.building != nil && zone.building.player_id != tickBase(actor).player_id {
			return "foundation_lost"
		}
	case defs.OrderType_UnitRepair:
		if building := r.buildings[uint64(order.target_id)]; building != nil {
			if _, cost, ok := repairHealAndCost(building, 1); ok && r.players[tickBase(actor).player_id].resources.affordFraction(cost) <= 0 {
				return "cannot_afford_repair"
			}
		}
	case defs.OrderType_UnitGarrison:
		return "garrison_full"
	}
	return "target_invalid"
}
