package risq

import "github.com/dgray001/gray_online/game/games/risq/internal/defs"

func (r *GameRisq) prepareTickActions(orderables []Orderable) {
	for _, actor := range orderables {
		base := tickBase(actor)
		action := base.tick_action
		if action == nil {
			continue
		}
		if !base.intent.hasIntent() {
			if action.execute.outcome != "blocked" {
				reason := "insufficient_stamina"
				if action.intent.available_stamina >= action.intent.min_cost {
					reason = "no_reachable_target"
					if action.intent.kind == "renew" {
						reason = "renew_complete"
					}
				}
				base.blockTickAction(reason)
			}
			continue
		}
		resolved := &TickAction{}
		r.snapshotIntent(resolved, base.intent)
		action.intent.resolution = tickResolution{
			kind: resolved.intent.kind, stamina_allocated: base.intent.intent_cost,
			sunk_cost: base.intent.sunk_cost,
		}
		if resolved.intent.kind != action.intent.kind {
			action.intent.resolution.reason = "melee_meeting"
			if resolved.intent.kind == "ungarrison" {
				action.intent.resolution.reason = "automatic_ungarrison"
			}
		}
		action.execute.kind = resolved.intent.kind
		action.execute.target, action.execute.target_location = resolved.intent.target, resolved.intent.target_location
		if resolved.intent.target.kind != "" {
			action.execute_target_visibility = resolved.target_visibility
		}
		if detail, ok := base.intent.detail.(*BuildingAttackIntent); ok {
			for _, attack := range detail.garrison_attacks {
				garrison := r.newTickAction(attack.unit, nil)
				garrison.order = &tickOrder{source: "building_garrison", source_actor: tickActorTarget(actor)}
				if action.order != nil {
					garrison.order.internal_id, garrison.order.order_type, garrison.order.target_id = action.order.internal_id, action.order.order_type, action.order.target_id
				}
				garrison.intent.kind = "attack"
				garrison.execute.kind = "attack"
				garrison.intent.min_cost, garrison.intent.max_cost = 1, unitTickStaminaCost
				garrison.intent.resolution = tickResolution{kind: "attack", stamina_allocated: attack.cost}
				garrison.targetSnapshot(tickActorTarget(detail.target), tickActorZone(detail.target))
				garrison.execute.target, garrison.execute.target_location = garrison.intent.target, garrison.intent.target_location
				garrison.execute_target_visibility = garrison.target_visibility
				attack.unit.garrison_tick_action = garrison
			}
		}
	}
}

func (b *orderableBase) startTickExecution() {
	if b.tick_action != nil {
		b.tick_action.execute.outcome = "executed"
	}
}

func (b *orderableBase) recordTickProgress(progress int) {
	if b.tick_action != nil {
		b.tick_action.execute.progress += progress
	}
}

func (r *GameRisq) recordCombatTick(attacker Attackable, target Attackable, damage float64, damage_type defs.AttackType) {
	if r.tick_history == nil || !r.tick_history.recording {
		return
	}
	actor := Orderable(attacker)
	action := tickBase(actor).tick_action
	if garrison, ok := attacker.(garrisonAttacker); ok {
		actor, action = garrison.RisqUnit, garrison.garrison_tick_action
	}
	r.tick_history.next_effect_id++
	effect := tickEffect{
		id: r.tick_history.next_effect_id, tick: r.tick_history.tick, damage_type: damage_type,
		kind: "damage", actor: tickActorTarget(actor), target: tickActorTarget(target), amount: damage,
		actor_visibility: tickZoneVisibility(tickActorZone(actor)), target_visibility: tickZoneVisibility(tickActorZone(target)),
	}
	if action != nil {
		effect.actor_visibility, effect.target_visibility = action.visibility, action.execute_target_visibility
		action.execute.effects = append(action.execute.effects, effect)
	}
	received := r.newTickAction(target, nil)
	received.visibility = effect.target_visibility
	if action != nil && action.execute.target_location != nil {
		received.intent.location = *action.execute.target_location
	}
	if proposal := tickBase(target).tick_action; proposal != nil {
		received.intent.available_stamina = proposal.intent.available_stamina
	}
	received.intent.kind, received.execute.outcome = "receive_damage", "executed"
	received.execute.kind = "receive_damage"
	received.execute.effects = append(received.execute.effects, effect)
	r.tick_history.effects = append(r.tick_history.effects, effect)
}

func (r *GameRisq) recordRepairTick(building *RisqBuilding, healing float64, cost defs.RisqResourceCost) {
	action := r.newTickAction(building, nil)
	if action == nil {
		return
	}
	action.intent.kind, action.execute.outcome = "repair_settlement", "executed"
	action.execute.kind = "repair_settlement"
	action.execute.healing, action.execute.cost = healing, cost
	r.tick_history.next_effect_id++
	effect := tickEffect{
		id: r.tick_history.next_effect_id, tick: r.tick_history.tick, kind: "heal",
		actor: tickActorTarget(building), target: tickActorTarget(building), amount: healing,
		actor_visibility: action.visibility, target_visibility: action.visibility,
	}
	action.execute.effects = append(action.execute.effects, effect)
	r.tick_history.effects = append(r.tick_history.effects, effect)
}

func (r *GameRisq) recordTickReceipt(actor Orderable, order *RisqOrder, kind, reason string) {
	if r.tick_history == nil || !r.tick_history.recording {
		return
	}
	r.tick_history.next_effect_id++
	r.tick_history.effects = append(r.tick_history.effects, tickEffect{
		id: r.tick_history.next_effect_id, tick: r.tick_history.tick, kind: kind, reason: reason,
		actor: tickActorTarget(actor), actor_visibility: tickZoneVisibility(tickActorZone(actor)),
		order: tickOrderValue(order), private: true,
	})
}

func (u *RisqUnit) recordGarrisonExecution(cost int, skipped bool) {
	if action := u.garrison_tick_action; action != nil {
		action.execute.stamina_spent = cost
		action.execute.outcome = "executed"
		if skipped {
			action.execute.outcome, action.execute.reason = "skipped", "own_action_preempted_garrison_attack"
		}
	}
}
