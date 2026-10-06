package risq

import (
	"cmp"
	"fmt"
	"iter"
	"maps"
	"slices"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
)

func (r *GameRisq) resolveActiveOrders() {
	util.DebugLog.Println("Resolving active orders")
	r.current_tick = 0
	r.beginTurnReports()
	for _, player := range r.players {
		for _, order := range receiptOrdered(player.active_orders) {
			if order.received {
				continue
			}
			order.internal_id = r.nextOrderInternalId()
			r.deliverOrder(order, player, false)
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
		intent_count -= r.resolveMeleeMeetings(orderables)
		if intent_count == 0 {
			r.metrics.beginTick(r, orderables)
			r.metrics.recordTick(r, r.current_tick+1)
			break
		}
		r.gather_allotments = computeGatherAllotments(orderables)
		r.garrison_allotments = computeGarrisonAllotments(orderables)
		r.construction_winners = computeConstructionWinners(orderables)
		r.foundation_ids = computeFoundationIds(r, r.construction_winners)
		r.population_slot_winners = computePopulationSlotWinners(r, orderables)
		r.repair_allotments = computeRepairAllotments(r, orderables)
		r.unit_creation_ids = computeUnitCreationIds(r, orderables)
		r.population_capped = computePopulationCapped(r)
		r.current_tick++
		r.metrics.beginTick(r, orderables)
		for _, o := range orderables {
			o.tickExecute(r)
		}
		r.resolveRepairs(orderables)
		r.metrics.recordTick(r, r.current_tick)
		// Applied after every actor's tickExecute so same-tick damage and healing net out
		// regardless of execution order, instead of racing on which lands first.
		for _, o := range orderables {
			if a, ok := o.(Attackable); ok {
				a.resolveHealthDelta(r)
			}
			if b, ok := o.(*RisqBuilding); ok {
				b.resolveRenew()
			}
		}
		for _, foundation_id := range r.foundation_ids {
			if building := r.buildings[foundation_id]; building != nil {
				building.resolveHealthDelta(r)
			}
		}
		// Applied last so a tech's combat bonus never affects the tick that finished researching it.
		for _, completion := range r.pending_tech_completions {
			r.completeResearch(r.players[completion.player_id], completion.tech_id)
		}
		r.pending_tech_completions = r.pending_tech_completions[:0]
		r.autoGatherCompletedBuildings()
		for _, building := range slices.SortedFunc(maps.Keys(r.unit_creation_ids), func(a, b *RisqBuilding) int {
			return cmp.Compare(r.unit_creation_ids[a], r.unit_creation_ids[b])
		}) {
			if building.gather_point != nil {
				unit := r.units[r.unit_creation_ids[building]]
				r.addSyntheticOrder(building.gather_point.resolveOrder(r, building, unit), r.players[building.player_id], false)
			}
		}
		r.logStateHash(fmt.Sprint(r.current_tick))
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
	r.endTurn()
	r.checkWinCondition()
	if !r.game.GameEnded() {
		r.startNextTurn()
	}
}

func (r *GameRisq) deliverOrder(order *RisqOrder, player *RisqPlayer, prepend bool) {
	player.report.orders.added++
	order.received = true
	order.turn_received = r.turn_number
	if order.order_type.IsPlayerOrder() {
		player.receivePlayerOrder(order, r)
		order.executed = true
		order.turn_resolved = r.turn_number
		return
	}
	if !r.deliverToSubjects(order, player, prepend) {
		order.cancelled = true
		order.turn_resolved = r.turn_number
	}
}

func (r *GameRisq) deliverToSubjects(order *RisqOrder, player *RisqPlayer, prepend bool) bool {
	accepted := false
	for _, subject_id := range slices.Sorted(maps.Keys(order.subjects)) {
		subject := order.subjects[subject_id]
		if !subject.orderReceivable(order, r) {
			order.rejectSubject(subject, player, "not receivable")
			continue
		}
		if err := subject.receiveOrder(order, r, prepend); err != nil {
			order.rejectSubject(subject, player, err.Error())
			continue
		}
		if order.clear_previous_orders {
			r.cancelPreviousOrders(subject, order)
		}
		accepted = true
	}
	return accepted
}

func (r *GameRisq) addSyntheticOrder(order *RisqOrder, player *RisqPlayer, prepend bool) {
	player.active_orders = append(player.active_orders, order)
	r.deliverOrder(order, player, prepend)
}

func (o *RisqOrder) rejectSubject(subject Orderable, player *RisqPlayer, reason string) {
	player.report.recordFailure(o.order_type, o.target_id, reason)
	if len(o.subjects) > 1 {
		delete(o.subjects, subject.internalId())
	}
}

func (r *GameRisq) cancelPreviousOrders(subject Orderable, keep *RisqOrder) {
	// cancelOrder mutates the subject's own active-orders slice in place, so range over a copy
	for _, other := range append([]*RisqOrder(nil), subject.activeOrders()...) {
		if other != keep && !other.executed && !other.cancelled && !other.order_type.IsClearImmune() {
			subject.cancelOrder(other, r)
		}
	}
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
			had_vision[player_id] = v >= defs.VisibilityPoor
			if v > defs.VisibilityFog {
				space.visibility[player_id] = defs.VisibilityFog
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
	for _, space := range r.allSpaces() {
		for player_id := range space.death_vision {
			if space.getVisibility(player_id) < defs.VisibilityPoor {
				space.visibility[player_id] = defs.VisibilityPoor
			}
		}
		clear(space.death_vision)
	}
	r.refreshVisionCaches(previously_visible)
}

// Refreshes a space's cache for a player if they had or now have vision of the space
func (r *GameRisq) refreshVisionCaches(previously_visible map[*RisqSpace]map[int]bool) {
	for _, space := range r.allSpaces() {
		for _, player := range r.players {
			player_id := player.player.Player_id
			if previously_visible[space][player_id] || space.getVisibility(player_id) >= defs.VisibilityPoor {
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
