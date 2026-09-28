package risq

import (
	"fmt"
	"os"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

func (r *GameRisq) requireGivingOrders(player *game.Player, kind string) bool {
	if !r.giving_orders {
		player.AddFailedUpdateShorthand(kind+"-failed", "Not currently giving orders")
		return false
	}
	return true
}

func (r *GameRisq) requireOrdersNotSubmitted(player *game.Player, kind string) bool {
	if r.players[player.Player_id].orders_submitted {
		player.AddFailedUpdateShorthand(kind+"-failed", "Orders already submitted")
		return false
	}
	return true
}

func (r *GameRisq) PlayerAction(action game.PlayerAction) {
	util.DebugLog.Println("player action:", action.Kind, action.Client_id, action.Ai_id, action.Action)
	player := r.game.AiPlayers[uint32(action.Ai_id)]
	if player == nil {
		player = r.game.Players[uint64(action.Client_id)]
	}
	if player == nil {
		fmt.Fprintln(os.Stderr, "Invalid client or ai id", action.Client_id, action.Ai_id)
		return
	}
	if r.game.GameEnded() {
		return
	}
	switch action.Kind {
	case "submit-orders":
		if !r.requireGivingOrders(player, "submit-orders") || !r.requireOrdersNotSubmitted(player, "submit-orders") {
			return
		}
		if !r.players[player.Player_id].canSubmitOrders() {
			player.AddFailedUpdateShorthand("submit-orders-failed", "Eliminated players cannot submit orders")
			return
		}
		orders, err := r.getOrdersFromPlayerAction(action.Action, player.Player_id)
		if err != nil {
			player.AddFailedUpdateShorthand("submit-orders-failed", err.Error())
			return
		}
		r.executeSubmitOrders(player.Player_id, orders)
	case "unsubmit-orders":
		if !r.requireGivingOrders(player, "unsubmit-orders") {
			return
		}
		if !r.players[player.Player_id].orders_submitted {
			player.AddFailedUpdateShorthand("unsubmit-orders-failed", "Orders not submitted")
			return
		}
		r.executeUnsubmitOrders(player.Player_id)
	case "set-unit-behavior":
		if !r.requireGivingOrders(player, "set-unit-behavior") || !r.requireOrdersNotSubmitted(player, "set-unit-behavior") {
			return
		}
		behavior, err := getUnitBehaviorFromPlayerAction(action.Action)
		if err != nil {
			player.AddFailedUpdateShorthand("set-unit-behavior-failed", err.Error())
			return
		}
		r.executeSetUnitBehavior(player.Player_id, behavior)
	case "set-building-behavior":
		if !r.requireGivingOrders(player, "set-building-behavior") || !r.requireOrdersNotSubmitted(player, "set-building-behavior") {
			return
		}
		behavior, err := getBuildingBehaviorFromPlayerAction(action.Action)
		if err != nil {
			player.AddFailedUpdateShorthand("set-building-behavior-failed", err.Error())
			return
		}
		r.executeSetBuildingBehavior(player.Player_id, behavior)
	case "set-gather-point":
		if !r.requireGivingOrders(player, "set-gather-point") || !r.requireOrdersNotSubmitted(player, "set-gather-point") {
			return
		}
		request, err := getGatherPointFromPlayerAction(action.Action)
		if err != nil {
			player.AddFailedUpdateShorthand("set-gather-point-failed", err.Error())
			return
		}
		r.executeSetGatherPoint(player.Player_id, request)
	default:
		fmt.Fprintln(os.Stderr, "Unknown game update type", action.Kind)
	}
}

func (r *GameRisq) executeSubmitOrders(player_id int, orders []defs.OrderFromFrontend) {
	util.DebugLog.Println("Executing submit orders for:", player_id, orders)
	player := r.players[player_id]
	new_orders := make([]*RisqOrder, 0, len(orders))
	for _, o := range orders {
		order_type := defs.OrderType(o.Order_type)
		subjects := make(map[uint64]Orderable)
		if order_type.IsUnitOrder() {
			for _, subject_id := range o.Subjects {
				subjects[subject_id] = r.players[o.Player_id].units[subject_id]
			}
		} else if order_type.IsBuildingOrder() {
			for _, subject_id := range o.Subjects {
				subjects[subject_id] = r.players[o.Player_id].buildings[subject_id]
			}
		}
		new_orders = append(new_orders, createRisqOrder(r.nextOrderInternalId(), order_type, player_id, subjects, o.Target_id, o.Clear_previous_orders))
	}
	player.active_orders = append(player.active_orders, new_orders...)
	player.orders_submitted = true
	all_orders_submitted := true
	for _, player := range r.players {
		if !player.orders_submitted && player.canSubmitOrders() {
			all_orders_submitted = false
		}
	}
	if all_orders_submitted {
		r.giving_orders = false
	}
	for _, player := range r.players {
		player.player.AddUpdate(&game.UpdateMessage{Kind: "submitted-orders", Content: gin.H{
			"player_id": player_id,
			"game":      r.toFrontendFor(player.player.Player_id, player.player.GetClientId(), false),
		}})
	}
	r.game.AddViewerUpdate(&game.UpdateMessage{Kind: "submitted-orders", Content: gin.H{
		"player_id": player_id,
		"game":      r.ToFrontend(0, true),
	}})
	if all_orders_submitted {
		r.resolveActiveOrders()
	}
}

func (r *GameRisq) executeUnsubmitOrders(player_id int) {
	util.DebugLog.Println("Executing unsubmit orders for:", player_id)
	player := r.players[player_id]
	player.orders_submitted = false
	kept := player.active_orders[:0]
	for _, order := range player.active_orders {
		if order.received {
			kept = append(kept, order)
		}
	}
	player.active_orders = kept
	for _, player := range r.players {
		player.player.AddUpdate(&game.UpdateMessage{Kind: "unsubmitted-orders", Content: gin.H{
			"player_id": player_id,
			"game":      r.toFrontendFor(player.player.Player_id, player.player.GetClientId(), false),
		}})
	}
	r.game.AddViewerUpdate(&game.UpdateMessage{Kind: "unsubmitted-orders", Content: gin.H{
		"player_id": player_id,
		"game":      r.ToFrontend(0, true),
	}})
}

// Resolves immediately; independent of the order system.
func (r *GameRisq) executeSetUnitBehavior(player_id int, behavior UnitBehaviorFromFrontend) {
	player := r.players[player_id]
	var target_priority []defs.TargetCategory
	if behavior.Target_priority != nil {
		target_priority = make([]defs.TargetCategory, 0, len(*behavior.Target_priority))
		for _, raw := range *behavior.Target_priority {
			cat := defs.TargetCategory(raw)
			if cat > defs.TargetCategory_NONE && cat < defs.TargetCategory_END {
				target_priority = append(target_priority, cat)
			}
		}
	}
	affected := make([]uint64, 0, len(behavior.Internal_ids))
	for _, internal_id := range behavior.Internal_ids {
		unit, ok := player.units[internal_id]
		if !ok || unit.unitType() == defs.UnitType_ECONOMIC {
			continue
		}
		if behavior.Stance != nil {
			stance := defs.UnitStance(*behavior.Stance)
			if stance > defs.UnitStance_NONE && stance < defs.UnitStance_END {
				unit.stance = stance
			}
		}
		if behavior.Interrupt_current != nil {
			unit.interrupt_current = *behavior.Interrupt_current
		}
		if behavior.Attack_back != nil {
			unit.attack_back = *behavior.Attack_back
		}
		if behavior.Target_priority != nil {
			unit.target_priority = target_priority
		}
		affected = append(affected, internal_id)
	}
	buildContent := func(ids []uint64) gin.H {
		content := gin.H{"internal_ids": ids}
		if behavior.Stance != nil {
			content["stance"] = *behavior.Stance
		}
		if behavior.Interrupt_current != nil {
			content["interrupt_current"] = *behavior.Interrupt_current
		}
		if behavior.Attack_back != nil {
			content["attack_back"] = *behavior.Attack_back
		}
		if behavior.Target_priority != nil {
			content["target_priority"] = targetCategoriesToInts(target_priority)
		}
		return content
	}
	player.player.AddUpdate(&game.UpdateMessage{Kind: "unit-behavior-set", Content: buildContent(affected)})
	for _, other := range r.players {
		if other.player.Player_id == player_id {
			continue
		}
		visible_ids := make([]uint64, 0)
		for _, internal_id := range affected {
			if unit := player.units[internal_id]; unit != nil && showOrdersTo(player_id, unit.zone, other.player.Player_id) {
				visible_ids = append(visible_ids, internal_id)
			}
		}
		if len(visible_ids) > 0 {
			other.player.AddUpdate(&game.UpdateMessage{Kind: "unit-behavior-set", Content: buildContent(visible_ids)})
		}
	}
}

// Resolves immediately; independent of the order system.
func (r *GameRisq) executeSetBuildingBehavior(player_id int, behavior BuildingBehaviorFromFrontend) {
	player := r.players[player_id]
	var target_priority []defs.TargetCategory
	if behavior.Target_priority != nil {
		target_priority = make([]defs.TargetCategory, 0, len(*behavior.Target_priority))
		for _, raw := range *behavior.Target_priority {
			cat := defs.TargetCategory(raw)
			if cat > defs.TargetCategory_NONE && cat < defs.TargetCategory_END {
				target_priority = append(target_priority, cat)
			}
		}
	}
	affected := make([]uint64, 0, len(behavior.Internal_ids))
	for _, internal_id := range behavior.Internal_ids {
		building, ok := player.buildings[internal_id]
		if !ok || defs.BuildingConfigs[building.building_id].Attack_type == defs.AttackType_NONE {
			continue
		}
		if behavior.Auto_attack != nil {
			building.auto_attack = *behavior.Auto_attack
		}
		if behavior.Interrupt_current != nil {
			building.interrupt_current = *behavior.Interrupt_current
		}
		if behavior.Target_priority != nil {
			building.target_priority = target_priority
		}
		affected = append(affected, internal_id)
	}
	buildContent := func(ids []uint64) gin.H {
		content := gin.H{"internal_ids": ids}
		if behavior.Auto_attack != nil {
			content["auto_attack"] = *behavior.Auto_attack
		}
		if behavior.Interrupt_current != nil {
			content["interrupt_current"] = *behavior.Interrupt_current
		}
		if behavior.Target_priority != nil {
			content["target_priority"] = targetCategoriesToInts(target_priority)
		}
		return content
	}
	player.player.AddUpdate(&game.UpdateMessage{Kind: "building-behavior-set", Content: buildContent(affected)})
	for _, other := range r.players {
		if other.player.Player_id == player_id {
			continue
		}
		visible_ids := make([]uint64, 0)
		for _, internal_id := range affected {
			if building := player.buildings[internal_id]; building != nil && showOrdersTo(player_id, building.zone, other.player.Player_id) {
				visible_ids = append(visible_ids, internal_id)
			}
		}
		if len(visible_ids) > 0 {
			other.player.AddUpdate(&game.UpdateMessage{Kind: "building-behavior-set", Content: buildContent(visible_ids)})
		}
	}
}

func (r *GameRisq) executeSetGatherPoint(player_id int, request GatherPointFromFrontend) {
	player := r.players[player_id]
	building, ok := player.buildings[request.Building_id]
	if !ok || !defs.BuildingConfigs[building.building_id].CanHaveGatherPoint() {
		return
	}
	if request.Clear {
		building.gather_point = nil
	} else {
		location_kind := RisqGatherPointLocationKind(request.Location_kind)
		if location_kind <= RisqGatherPointLocationKind_NONE || location_kind >= RisqGatherPointLocationKind_END {
			return
		}
		switch location_kind {
		case RisqGatherPointLocationKind_SPACE:
			if invertSpaceKey(uint(request.Location_id), r) == nil {
				return
			}
		case RisqGatherPointLocationKind_ZONE:
			if _, zone := invertZoneKey(uint(request.Location_id), r); zone == nil {
				return
			}
		}
		object_type := RisqGatherObjectType(request.Object_type)
		if object_type >= RisqGatherObjectType_END {
			return
		}
		building.gather_point = &RisqGatherPoint{
			location_kind: location_kind,
			location_id:   request.Location_id,
			object_type:   object_type,
			object_id:     request.Object_id,
		}
	}
	content := gin.H{"building_id": request.Building_id}
	if building.gather_point != nil {
		content["gather_point"] = building.gather_point.toFrontend()
	}
	player.player.AddUpdate(&game.UpdateMessage{Kind: "gather-point-set", Content: content})
}
