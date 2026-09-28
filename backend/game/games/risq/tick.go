package risq

import (
	"fmt"
	"iter"

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
			player.report.orders.added++
			order.received = true
			order.turn_received = r.turn_number
			if order.order_type.IsPlayerOrder() {
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
						if !other.executed && !other.cancelled && !other.order_type.IsClearImmune() {
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
		r.foundation_ids = computeFoundationIds(r, r.construction_winners)
		r.population_slot_winners = computePopulationSlotWinners(r, orderables)
		r.repair_allotments = computeRepairAllotments(r, orderables)
		r.unit_creation_ids = computeUnitCreationIds(r, orderables)
		r.population_capped = computePopulationCapped(r)
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
			if b, ok := o.(*RisqBuilding); ok {
				b.resolveRenew()
			}
		}
		// Applied last so a tech's combat bonus never affects the tick that finished researching it.
		for _, completion := range r.pending_tech_completions {
			r.completeResearch(r.players[completion.player_id], completion.tech_id)
		}
		r.pending_tech_completions = r.pending_tech_completions[:0]
		r.autoGatherCompletedBuildings()
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
