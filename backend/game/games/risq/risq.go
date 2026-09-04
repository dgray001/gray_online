package risq

import (
	"fmt"
	"iter"
	"os"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/gin-gonic/gin"
)

/*
   ================
   >>>>> RISQ <<<<<
   ================

   Objective: Build your empire and conquer the world!
   Description: Strategy board game with simultaneous turn resolution, hexgonal
     map, complex deterministic mechanics (no randomness after map generation),
  	 resource gathering, empire-building, complex combat, and medieval themes.
*/

type GameRisq struct {
	game                      *game.GameBase
	players                   []*RisqPlayer
	board_size                uint16
	population_limit          uint16
	spaces                    [][]*RisqSpace
	next_resource_internal_id uint64
	next_building_internal_id uint64
	next_unit_internal_id     uint64
	next_order_internal_id    uint64
	turn_number               uint16
	// True if waiting for players to give orders and false if resolving active orders
	giving_orders bool
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

func (r *GameRisq) coordinateToIndex(c *game_utils.Coordinate2D) *game_utils.Coordinate2D {
	return &game_utils.Coordinate2D{
		X: c.Y + int(r.board_size),
		Y: c.X - max(-int(r.board_size), -(int(r.board_size)+c.Y)),
	}
}

func (r *GameRisq) getSpace(c *game_utils.Coordinate2D) *RisqSpace {
	index := r.coordinateToIndex(c)
	if index.X < 0 || index.X >= len(r.spaces) {
		return nil
	}
	row := r.spaces[index.X]
	if index.Y < 0 || index.Y >= len(row) {
		return nil
	}
	return row[index.Y]
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
	r.giving_orders = true
	for _, player := range r.players {
		player.player.AddUpdate(&game.UpdateMessage{Kind: "start-turn", Content: gin.H{
			"game": r.ToFrontend(player.player.GetClientId(), false),
		}})
	}
	r.game.AddViewerUpdate(&game.UpdateMessage{Kind: "start-turn", Content: gin.H{
		"game": r.ToFrontend(0, true),
	}})
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

func (r *GameRisq) PlayerAction(action game.PlayerAction) {
	fmt.Println("player action:", action.Kind, action.Client_id, action.Action)
	player := r.game.Players[uint64(action.Client_id)]
	if player == nil {
		fmt.Fprintln(os.Stderr, "Invalid client id", action.Client_id)
		return
	}
	switch action.Kind {
	case "submit-orders":
		if !r.giving_orders {
			player.AddFailedUpdateShorthand("submit-orders-failed", "Not currently giving orders")
			return
		}
		if r.players[player.Player_id].orders_submitted {
			player.AddFailedUpdateShorthand("submit-orders-failed", "Orders already submitted")
			return
		}
		orders, err := r.getOrdersFromPlayerAction(action.Action)
		if err != nil {
			player.AddFailedUpdateShorthand("submit-orders-failed", err.Error())
			return
		}
		r.executeSubmitOrders(player.Player_id, orders)
	case "unsubmit-orders":
		if !r.giving_orders {
			player.AddFailedUpdateShorthand("unsubmit-orders-failed", "Not currently giving orders")
			return
		}
		if !r.players[player.Player_id].orders_submitted {
			player.AddFailedUpdateShorthand("submit-orders-failed", "Orders not submitted")
			return
		}
		r.executeUnsubmitOrders(player.Player_id)
	default:
		fmt.Fprintln(os.Stderr, "Unknown game update type", action.Kind)
	}
}

func (r *GameRisq) executeSubmitOrders(player_id int, orders []OrderFromFrontend) {
	fmt.Println("Executing submit orders for:", player_id, orders)
	player := r.players[player_id]
	new_orders := make([]*RisqOrder, 0, len(orders))
	for _, o := range orders {
		order_type := OrderType(o.Order_type)
		subjects := make([]Orderable, 0)
		if order_type.isUnitOrder() {
			for _, subject_id := range o.Subjects {
				subjects = append(subjects, r.players[o.Player_id].units[subject_id])
			}
		} else if order_type.isBuildingOrder() {
			for _, subject_id := range o.Subjects {
				subjects = append(subjects, r.players[o.Player_id].buildings[subject_id])
			}
		}
		new_orders = append(new_orders, createRisqOrder(r.nextOrderInternalId(), order_type, player_id, subjects, o.Target_id, o.Clear_previous_orders))
	}
	player.active_orders = append(player.active_orders, new_orders...)
	player.orders_submitted = true
	all_orders_submitted := true
	for _, player := range r.players {
		if !player.orders_submitted {
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
	fmt.Println("Executing unsubmit orders for:", player_id)
	player := r.players[player_id]
	player.orders_submitted = false
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

func (r *GameRisq) resolveActiveOrders() {
	fmt.Println("Resolving active orders")
	r.giving_orders = false
	for _, player := range r.players {
		for _, order := range player.active_orders {
			if order.received {
				continue
			}
			order.received = true
			order.turn_received = r.turn_number
			if order.order_type.isPlayerOrder() {
				player.receivePlayerOrder(order, r)
				order.executed = true
				order.turn_executed = r.turn_number
				continue
			}
			for _, subject := range order.subjects {
				if !subject.orderReceivable(order, r) {
					continue
				}
				if order.clear_previous_orders {
					for _, other := range player.active_orders {
						if other == order || other.executed || other.cancelled {
							continue
						}
						for _, other_subject := range other.subjects {
							if other_subject == subject {
								subject.cancelOrder(other, r)
								break
							}
						}
					}
				}
				subject.receiveOrder(order, r)
			}
		}
	}
	for {
		no_intents := true
		for o := range r.allOrderables() {
			if o.tickIntent(r) {
				no_intents = false
			}
		}
		if no_intents {
			break
		}
		for o := range r.allOrderables() {
			o.tickExecute(r)
		}
	}
	for _, player := range r.players {
		kept := player.active_orders[:0]
		for _, order := range player.active_orders {
			if order.received && !order.executed && !order.cancelled {
				kept = append(kept, order)
			}
		}
		player.active_orders = kept
	}
	r.startNextTurn()
}

func (r *GameRisq) recalculateVision() {
	for _, row := range r.spaces {
		for _, space := range row {
			for player_id, v := range space.visibility {
				if v > VisibilityFog {
					space.visibility[player_id] = VisibilityFog
				}
			}
		}
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
	r.refreshVisionCaches()
}

func (r *GameRisq) refreshVisionCaches() {
	for _, row := range r.spaces {
		for _, space := range row {
			for _, player := range r.players {
				player_id := player.player.Player_id
				if space.getVisibility(player_id) >= VisibilityPoor {
					space.refreshCache(player_id)
				}
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
			spaces_row = append(spaces_row, space.toFrontend(player_id, is_viewer))
		}
		spaces = append(spaces, spaces_row)
	}
	game["spaces"] = spaces
	return game
}
