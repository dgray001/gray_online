package risq

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
)

type Orderable interface {
	toFrontend(viewer_player_id int) gin.H
	isDeleted() bool
	internalId() uint64
	OrderableType() defs.OrderableType
	activeOrders() []*RisqOrder
	refreshStamina()
	orderReceivable(o *RisqOrder, risq *GameRisq) bool
	// Returns failure reason if order was not received
	receiveOrder(o *RisqOrder, risq *GameRisq) error
	cancelOrder(o *RisqOrder, risq *GameRisq)
	orderStatus(o *RisqOrder, risq *GameRisq) OrderStatus
	tickIntent(risq *GameRisq) bool
	tickExecute(risq *GameRisq)
	// Removes this orderable from all maps/zones/orders once deleted; called once per player per turn
	cleanupDeleted(risq *GameRisq)
}

// Resolves orders naming internal_id as a subject; shared by RisqUnit/RisqBuilding's cleanupDeleted
func resolveOrdersOnDeath(risq *GameRisq, queue *RisqOrderQueue, internal_id uint64, self_delete_order_type defs.OrderType) {
	for _, o := range queue.active_orders {
		if len(o.subjects) > 1 {
			delete(o.subjects, internal_id)
			continue
		}
		if o.order_type == self_delete_order_type {
			o.executed = true
		} else {
			o.cancelled = true
		}
		o.turn_resolved = risq.turn_number
	}
	queue.active_orders = nil
}

type RisqOrderQueue struct {
	active_orders []*RisqOrder
	past_orders   []*RisqOrder
}

type OrderStatus uint8

const (
	OrderStatus_NONE OrderStatus = iota
	OrderStatus_InProgress
	OrderStatus_Executed
	OrderStatus_Cancelled
)

type RisqOrder struct {
	internal_id uint64
	player_id   int
	// The targets this order is effecting, keyed by subject internal id for O(1) removal
	subjects   map[uint64]Orderable
	order_type defs.OrderType
	// What the order is targeting (could be a space, a unit, or a technology)
	target_id             int64
	clear_previous_orders bool
	received              bool
	executed              bool
	cancelled             bool
	turn_received         uint16
	turn_resolved         uint16
}

func createRisqOrder(internal_id uint64, order_type defs.OrderType, player_id int, subjects map[uint64]Orderable, target_id int64, clear_previous_orders bool) *RisqOrder {
	order := RisqOrder{
		internal_id:           internal_id,
		player_id:             player_id,
		subjects:              subjects,
		order_type:            order_type,
		target_id:             target_id,
		clear_previous_orders: clear_previous_orders,
	}
	return &order
}

func (o *RisqOrder) toFrontend() gin.H {
	order := gin.H{
		"internal_id":           o.internal_id,
		"player_id":             o.player_id,
		"order_type":            o.order_type,
		"target_id":             o.target_id,
		"turn_received":         o.turn_received,
		"turn_resolved":         o.turn_resolved,
		"clear_previous_orders": o.clear_previous_orders,
	}
	subjects := make([]uint64, 0)
	for _, subject := range o.subjects {
		if subject != nil && !subject.isDeleted() {
			subjects = append(subjects, subject.internalId())
		}
	}
	order["subjects"] = subjects
	return order
}

// Refund orders come first so a same-batch order can spend what they free up, regardless of submission order
func receiptOrdered(orders []*RisqOrder) []*RisqOrder {
	ordered := make([]*RisqOrder, 0, len(orders))
	for _, refund_pass := range []bool{true, false} {
		for _, o := range orders {
			if o.order_type.IsRefundOrder() == refund_pass {
				ordered = append(ordered, o)
			}
		}
	}
	return ordered
}

func createRisqOrderQueue() RisqOrderQueue {
	return RisqOrderQueue{
		active_orders: make([]*RisqOrder, 0),
		past_orders:   make([]*RisqOrder, 0),
	}
}

func (q *RisqOrderQueue) receiveOrder(o *RisqOrder) {
	q.past_orders = append(q.past_orders, o)
	q.active_orders = append(q.active_orders, o)
}

// Removes the order with this internal id from the queue; returns it, or nil if not found
func (q *RisqOrderQueue) removeOrder(internal_id uint64) *RisqOrder {
	for i, order := range q.active_orders {
		if order.internal_id == internal_id {
			q.active_orders = append(q.active_orders[:i], q.active_orders[i+1:]...)
			return order
		}
	}
	return nil
}

func (q *RisqOrderQueue) nextOrder(orderable Orderable, risq *GameRisq) *RisqOrder {
	for len(q.active_orders) > 0 {
		o := q.active_orders[0]
		switch orderable.orderStatus(o, risq) {
		case OrderStatus_InProgress:
			return o
		case OrderStatus_Cancelled:
			if len(o.subjects) > 1 {
				delete(o.subjects, orderable.internalId())
			} else {
				o.cancelled = true
				o.turn_resolved = risq.turn_number
			}
		default:
			if len(o.subjects) > 1 {
				delete(o.subjects, orderable.internalId())
			} else {
				o.executed = true
				o.turn_resolved = risq.turn_number
			}
		}
		q.active_orders = q.active_orders[1:]
	}
	return nil
}
