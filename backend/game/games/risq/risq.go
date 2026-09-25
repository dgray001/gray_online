package risq

import (
	"fmt"
	"iter"
	"math/rand"
	"os"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

/*
   ================
   >>>>> RISQ <<<<<
   ================

   Objective: Build your empire and conquer the world!
   Description: Strategy board game with simultaneous turn resolution, hexagonal
     map, complex deterministic mechanics (no randomness after map generation),
     resource gathering, empire-building, complex combat, and medieval themes.
*/

type GameRisq struct {
	game                      *game.GameBase
	players                   []*RisqPlayer
	board_size                uint16
	population_limit          uint16
	spaces                    [][]*RisqSpace
	units                     map[uint64]*RisqUnit
	buildings                 map[uint64]*RisqBuilding
	next_resource_internal_id uint64
	next_building_internal_id uint64
	next_unit_internal_id     uint64
	next_order_internal_id    uint64
	turn_number               uint16
	current_tick              uint16
	// True if waiting for players to give orders and false if resolving active orders
	giving_orders bool
	// Recomputed each tick: contested resources are water-filled instead of first-come-first-served
	gather_allotments map[*RisqUnit]float64
	// Recomputed each tick: settles which simultaneous garrison attempts get a building's remaining slots
	garrison_allotments map[*RisqUnit]bool
	// Recomputed each tick: settles which unit founds a new building when several race for the same empty zone
	construction_winners map[*RisqZone]uint64
	// Recomputed each tick: settles which building's unit production completes when several race for the last population slots
	population_slot_winners map[*RisqBuilding]bool
	// Recomputed each tick: a player's simultaneous repairs are water-filled per resource category
	// instead of whichever executes first draining the shared balance
	repair_allotments map[*RisqUnit]float64
	// Zones whose cosmetic terrain_override should clear at the start of cleanupDeleted's NEXT call,
	// so a destroyed building's override is still visible for the turn following its death
	pending_terrain_clears []*RisqZone
	// Techs finishing production this tick; applied after this tick's health deltas so a tech's
	// combat bonus never affects the same tick's combat, only the next one
	pending_tech_completions []techCompletion
	regions                  []*RisqRegion
	// owned by this game only, never the shared global source, so concurrent AI goroutines can't race it
	rng *rand.Rand
}

func (r *GameRisq) nextResourceInternalId() uint64 {
	r.next_resource_internal_id++
	return r.next_resource_internal_id
}

func (r *GameRisq) nextBuildingInternalId() uint64 {
	r.next_building_internal_id++
	return r.next_building_internal_id
}

func (r *GameRisq) nextUnitInternalId() uint64 {
	r.next_unit_internal_id++
	return r.next_unit_internal_id
}

func (r *GameRisq) nextOrderInternalId() uint64 {
	r.next_order_internal_id++
	return r.next_order_internal_id
}

func (r *GameRisq) GetBase() *game.GameBase {
	return r.game
}

func (r *GameRisq) StartGame() {
	r.startNextTurn()
}

func (r *GameRisq) startNextTurn() {
	r.turn_number++
	for _, player := range r.players {
		player.orders_submitted = false
	}
	for o := range r.allOrderables() {
		o.refreshStamina()
	}
	r.recalculateOwnership()
	r.recalculateVision()
	r.refreshScores()
	r.updateEliminated()
	r.finalizeTurnReports()
	r.giving_orders = true
	for _, player := range r.players {
		player.player.AddUpdate(&game.UpdateMessage{Kind: "start-turn", Content: gin.H{
			"game": r.ToFrontend(player.player.GetClientId(), false),
		}})
	}
	r.game.AddViewerUpdate(&game.UpdateMessage{Kind: "start-turn", Content: gin.H{
		"game": r.ToFrontend(0, true),
	}})
	r.checkWinCondition()
}

func (r *GameRisq) updateEliminated() {
	for _, player := range r.players {
		if player.eliminated {
			continue
		}
		if len(player.units) == 0 && len(player.buildings) == 0 {
			player.eliminated = true
			player.report.recordEliminated()
			if !player.player.IsHumanPlayer() {
				close(player.ai_stop)
			}
		}
	}
}

func (r *GameRisq) checkWinCondition() {
	if len(r.players) < 2 {
		return
	}
	var remaining []*RisqPlayer
	for _, player := range r.players {
		if !player.eliminated {
			remaining = append(remaining, player)
		}
	}
	if len(remaining) > 1 {
		return
	}
	if len(remaining) == 1 {
		r.game.EndGame(fmt.Sprintf("%s wins!", remaining[0].player.GetNickname()))
	} else {
		r.game.EndGame("No players remaining")
	}
}

func (r *GameRisq) Valid() bool {
	if r.game == nil {
		return false
	}
	for _, player := range r.players {
		if !player.valid() {
			return false
		}
	}
	return true
}

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

func (r *GameRisq) executeSubmitOrders(player_id int, orders []OrderFromFrontend) {
	util.DebugLog.Println("Executing submit orders for:", player_id, orders)
	player := r.players[player_id]
	new_orders := make([]*RisqOrder, 0, len(orders))
	for _, o := range orders {
		order_type := OrderType(o.Order_type)
		subjects := make(map[uint64]Orderable)
		if order_type.isUnitOrder() {
			for _, subject_id := range o.Subjects {
				subjects[subject_id] = r.players[o.Player_id].units[subject_id]
			}
		} else if order_type.isBuildingOrder() {
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
			"game":      r.ToFrontend(player.player.GetClientId(), false),
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
			"game":      r.ToFrontend(player.player.GetClientId(), false),
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
	var target_priority []TargetCategory
	if behavior.Target_priority != nil {
		target_priority = make([]TargetCategory, 0, len(*behavior.Target_priority))
		for _, raw := range *behavior.Target_priority {
			cat := TargetCategory(raw)
			if cat > TargetCategory_NONE && cat < TargetCategory_END {
				target_priority = append(target_priority, cat)
			}
		}
	}
	affected := make([]uint64, 0, len(behavior.Internal_ids))
	for _, internal_id := range behavior.Internal_ids {
		unit, ok := player.units[internal_id]
		if !ok || unit.unitType() == UnitType_ECONOMIC {
			continue
		}
		if behavior.Stance != nil {
			stance := UnitStance(*behavior.Stance)
			if stance > UnitStance_NONE && stance < UnitStance_END {
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
			content["target_priority"] = *behavior.Target_priority
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
	var target_priority []TargetCategory
	if behavior.Target_priority != nil {
		target_priority = make([]TargetCategory, 0, len(*behavior.Target_priority))
		for _, raw := range *behavior.Target_priority {
			cat := TargetCategory(raw)
			if cat > TargetCategory_NONE && cat < TargetCategory_END {
				target_priority = append(target_priority, cat)
			}
		}
	}
	affected := make([]uint64, 0, len(behavior.Internal_ids))
	for _, internal_id := range behavior.Internal_ids {
		building, ok := player.buildings[internal_id]
		if !ok || buildingConfigs[building.building_id].attack_type == AttackType_NONE {
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
			content["target_priority"] = *behavior.Target_priority
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
	if !ok || !buildingConfigs[building.building_id].canHaveGatherPoint() {
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

func (r *GameRisq) resolveActiveOrders() {
	util.DebugLog.Println("Resolving active orders")
	r.current_tick = 0
	r.beginTurnReports()
	for _, player := range r.players {
		for _, order := range player.active_orders {
			if order.received {
				continue
			}
			player.report.orders.added++
			order.received = true
			order.turn_received = r.turn_number
			if order.order_type.isPlayerOrder() {
				player.receivePlayerOrder(order, r)
				order.executed = true
				order.turn_resolved = r.turn_number
				continue
			}
			accepted := false
			for _, subject := range order.subjects {
				if !subject.orderReceivable(order, r) {
					player.report.recordFailure(order.order_type, order.target_id, "not receivable")
					if len(order.subjects) > 1 {
						delete(order.subjects, subject.internalId())
					}
					continue
				}
				if order.clear_previous_orders {
					// cancelOrder mutates the subject's own active-orders slice in place, so range over a copy
					previous := append([]*RisqOrder(nil), subject.activeOrders()...)
					for _, other := range previous {
						if !other.executed && !other.cancelled {
							subject.cancelOrder(other, r)
						}
					}
				}
				if err := subject.receiveOrder(order, r); err != nil {
					player.report.recordFailure(order.order_type, order.target_id, err.Error())
					if len(order.subjects) > 1 {
						delete(order.subjects, subject.internalId())
					}
					continue
				}
				accepted = true
			}
			if !accepted {
				order.cancelled = true
				order.turn_resolved = r.turn_number
			}
		}
	}
	for {
		orderables := make([]Orderable, 0)
		for o := range r.allOrderables() {
			orderables = append(orderables, o)
		}
		intent_count := 0
		for _, o := range orderables {
			if o.tickIntent(r) {
				intent_count++
			}
		}
		if intent_count == 0 {
			break
		}
		r.gather_allotments = computeGatherAllotments(orderables)
		r.garrison_allotments = computeGarrisonAllotments(orderables)
		r.construction_winners = computeConstructionWinners(orderables)
		r.population_slot_winners = computePopulationSlotWinners(r, orderables)
		r.repair_allotments = computeRepairAllotments(r, orderables)
		r.current_tick++
		for _, o := range orderables {
			o.tickExecute(r)
		}
		// Applied after every actor's tickExecute so same-tick damage and healing net out
		// regardless of execution order, instead of racing on which lands first.
		for _, o := range orderables {
			if a, ok := o.(Attackable); ok {
				a.resolveHealthDelta(r)
			}
		}
		// Applied last so a tech's combat bonus never affects the tick that finished researching it.
		for _, completion := range r.pending_tech_completions {
			r.completeResearch(r.players[completion.player_id], completion.tech_id)
		}
		r.pending_tech_completions = r.pending_tech_completions[:0]
	}
	r.cleanupDeleted()
	for _, player := range r.players {
		kept := player.active_orders[:0]
		for _, order := range player.active_orders {
			if order.executed {
				player.report.orders.executed++
			} else if order.cancelled {
				player.report.orders.cancelled++
			}
			if order.received && !order.executed && !order.cancelled {
				kept = append(kept, order)
			}
		}
		player.active_orders = kept
		player.report.orders.active = len(kept)
	}
	r.startNextTurn()
}

func (r *GameRisq) cancelPendingTerrainClear(zone *RisqZone) {
	for i, z := range r.pending_terrain_clears {
		if z == zone {
			r.pending_terrain_clears = append(r.pending_terrain_clears[:i], r.pending_terrain_clears[i+1:]...)
			return
		}
	}
}

func (r *GameRisq) cleanupDeleted() {
	for _, zone := range r.pending_terrain_clears {
		zone.terrain_override = 0
	}
	r.pending_terrain_clears = r.pending_terrain_clears[:0]
	for _, player := range r.players {
		orderables := make([]Orderable, 0, len(player.units)+len(player.buildings))
		for _, u := range player.units {
			orderables = append(orderables, u)
		}
		for _, b := range player.buildings {
			orderables = append(orderables, b)
		}
		for _, o := range orderables {
			if o.isDeleted() {
				o.cleanupDeleted(r)
			}
		}
	}
}

func (r *GameRisq) recalculateVision() {
	previously_visible := make(map[*RisqSpace]map[int]bool)
	for _, space := range r.allSpaces() {
		had_vision := make(map[int]bool, len(space.visibility))
		for player_id, v := range space.visibility {
			had_vision[player_id] = v >= VisibilityPoor
			if v > VisibilityFog {
				space.visibility[player_id] = VisibilityFog
			}
		}
		previously_visible[space] = had_vision
	}
	for _, player := range r.players {
		for _, unit := range player.units {
			if unit.deleted || unit.zone == nil {
				continue
			}
			unit.zone.space.addVision(unit.vision(), unit.zone, unit.player_id)
		}
		for _, building := range player.buildings {
			if building.deleted || building.zone == nil {
				continue
			}
			building.zone.space.addVision(building.vision(), building.zone, building.player_id)
		}
	}
	r.refreshVisionCaches(previously_visible)
}

// Refreshes a space's cache for a player if they had or now have vision of the space
func (r *GameRisq) refreshVisionCaches(previously_visible map[*RisqSpace]map[int]bool) {
	for _, space := range r.allSpaces() {
		for _, player := range r.players {
			player_id := player.player.Player_id
			if previously_visible[space][player_id] || space.getVisibility(player_id) >= VisibilityPoor {
				space.refreshCache(player_id)
			}
		}
	}
}

func (r *GameRisq) allOrderables() iter.Seq[Orderable] {
	return func(yield func(Orderable) bool) {
		for _, player := range r.players {
			for o := range player.allOrderables() {
				if !yield(o) {
					return
				}
			}
		}
	}
}

func (r *GameRisq) PlayerDisconnected(client_id uint64) {
}

func (r *GameRisq) PlayerReconnected(client_id uint64) {
}

func (r *GameRisq) ToFrontend(client_id uint64, is_viewer bool) gin.H {
	game := gin.H{
		"board_size":       r.board_size,
		"population_limit": r.population_limit,
		"turn_number":      r.turn_number,
		"giving_orders":    r.giving_orders,
	}
	if r.game != nil {
		game["game_base"] = r.game.ToFrontend(client_id, is_viewer)
	}
	player_id := -1
	if !is_viewer {
		for id, player := range r.players {
			if player != nil && player.player.GetClientId() == client_id {
				player_id = id
				break
			}
		}
	}
	players := []gin.H{}
	for _, player := range r.players {
		if player != nil {
			players = append(players, player.toFrontend(player_id))
		}
	}
	game["players"] = players
	spaces := [][]gin.H{}
	for _, row := range r.spaces {
		spaces_row := []gin.H{}
		for _, space := range row {
			if space == nil {
				spaces_row = append(spaces_row, nil)
				continue
			}
			spaces_row = append(spaces_row, space.toFrontend(player_id, is_viewer))
		}
		spaces = append(spaces, spaces_row)
	}
	game["spaces"] = spaces
	regions := []gin.H{}
	for _, region := range r.regions {
		if reg := region.toFrontend(r, player_id); reg != nil {
			regions = append(regions, reg)
		}
	}
	game["regions"] = regions
	return game
}
