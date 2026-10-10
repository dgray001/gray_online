package risq

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
)

func (v tickVisibility) inspect(player_id, owner int) bool {
	return player_id >= 0 && (player_id == owner || v[player_id] >= defs.VisibilitySpy)
}

func (v tickVisibility) see(player_id, owner int) bool {
	return player_id >= 0 && (player_id == owner || v[player_id] >= defs.VisibilityGood)
}

func (l tickLocation) toFrontend() gin.H {
	result := gin.H{"space": l.space.ToFrontend(), "zone": l.zone.ToFrontend()}
	if l.garrisoned_in != 0 {
		result["garrisoned_in"] = l.garrisoned_in
	}
	return result
}

func (t tickTarget) toFrontend() gin.H {
	if t.kind == "" {
		return gin.H{}
	}
	return gin.H{"kind": t.kind, "internal_id": t.internal_id, "player_id": t.player_id}
}

func (o tickOrder) toFrontend() gin.H {
	result := gin.H{"order_type": o.order_type, "target_id": o.target_id, "source": o.source}
	if o.internal_id != 0 {
		result["internal_id"] = o.internal_id
	}
	if o.source_actor.kind != "" {
		result["source_actor"] = o.source_actor.toFrontend()
	}
	return result
}

func (e tickEffect) toFrontend(player_id int) gin.H {
	if e.private && player_id != e.actor.player_id {
		return nil
	}
	if !e.actor_visibility.inspect(player_id, e.actor.player_id) && !e.target_visibility.see(player_id, e.target.player_id) {
		return nil
	}
	result := gin.H{"id": e.id, "tick": e.tick, "kind": e.kind, "amount": e.amount, "damage_type": e.damage_type}
	if e.order != nil {
		result["order"], result["reason"] = e.order.toFrontend(), e.reason
	}
	if e.actor_visibility.see(player_id, e.actor.player_id) {
		result["actor"] = e.actor.toFrontend()
	}
	if e.target_visibility.see(player_id, e.target.player_id) {
		result["target"] = e.target.toFrontend()
	}
	return result
}

func (a *TickAction) toFrontend(player_id int) gin.H {
	if !a.visibility.inspect(player_id, a.player_id) {
		return nil
	}
	resolution := a.intent.resolution
	intent := gin.H{
		"kind": a.intent.kind, "location": a.intent.location.toFrontend(),
		"available_stamina": a.intent.available_stamina, "min_cost": a.intent.min_cost, "max_cost": a.intent.max_cost,
		"resolution": gin.H{"kind": resolution.kind, "reason": resolution.reason, "stamina_allocated": resolution.stamina_allocated, "sunk_cost": resolution.sunk_cost},
	}
	if a.intent.destination != nil {
		intent["destination"] = a.intent.destination.toFrontend()
	}
	if a.intent.item_id != 0 {
		intent["item_id"], intent["producible_kind"] = a.intent.item_id, a.intent.producible_kind
	}
	execution := gin.H{
		"kind":    a.execute.kind,
		"outcome": a.execute.outcome, "reason": a.execute.reason, "stamina_spent": a.execute.stamina_spent,
		"progress": a.execute.progress, "gathered": a.execute.gathered, "healing": a.execute.healing,
		"food_spent": a.execute.cost.Food, "wood_spent": a.execute.cost.Wood,
		"stone_spent": a.execute.cost.Stone, "gold_spent": a.execute.cost.Gold,
	}
	if player_id == a.player_id || a.target_visibility.see(player_id, a.intent.target.player_id) {
		intent["target"] = a.intent.target.toFrontend()
		if a.intent.target_location != nil {
			intent["target_location"] = a.intent.target_location.toFrontend()
		}
	}
	if player_id == a.player_id || a.execute_target_visibility.see(player_id, a.execute.target.player_id) {
		execution["target"] = a.execute.target.toFrontend()
		if a.execute.target_location != nil {
			execution["target_location"] = a.execute.target_location.toFrontend()
		}
	}
	effects := []gin.H{}
	for _, effect := range a.execute.effects {
		if value := effect.toFrontend(player_id); value != nil {
			effects = append(effects, value)
		}
	}
	execution["effects"] = effects
	result := gin.H{"tick": a.tick, "sequence": a.sequence, "intent": intent, "execute": execution}
	if a.order != nil {
		result["order"] = a.order.toFrontend()
	}
	return result
}

func tickActionsToFrontend(actions []*TickAction, player_id int) []gin.H {
	result := []gin.H{}
	for _, action := range actions {
		if value := action.toFrontend(player_id); value != nil {
			result = append(result, value)
		}
	}
	return result
}
