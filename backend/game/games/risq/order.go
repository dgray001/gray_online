package risq

import (
	"maps"
	"slices"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
)

type Orderable interface {
	toFrontend(viewer_player_id int, with_history bool) gin.H
	isDeleted() bool
	internalId() uint64
	OrderableType() defs.OrderableType
	activeOrders() []*RisqOrder
	refreshStamina()
	orderReceivable(o *RisqOrder, risq *GameRisq) bool
	// Returns failure reason if order was not received
	receiveOrder(o *RisqOrder, risq *GameRisq, prepend bool) error
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
		o.resolveSubject(internal_id, o.order_type == self_delete_order_type, risq.turn_number)
	}
	queue.active_orders = nil
}

type RisqOrderQueue struct {
	active_orders []*RisqOrder
}

func (r *GameRisq) clearAllOrders() {
	for _, player := range r.players {
		for entity := range player.allOrderables() {
			for _, order := range append([]*RisqOrder(nil), entity.activeOrders()...) {
				entity.cancelOrder(order, r)
			}
			if unit, ok := entity.(*RisqUnit); ok {
				unit.move_path = nil
				unit.half_move = nil
			}
		}
		player.active_orders = nil
		player.unreceived_orders = nil
		if player.report != nil {
			player.report.orders.active = 0
		}
	}
}

type OrderStatus uint8

const (
	OrderStatus_NONE OrderStatus = iota
	OrderStatus_InProgress
	OrderStatus_Executed
	OrderStatus_Cancelled
)

type RisqOrder struct {
	tick_source string

	internal_id uint64
	player_id   int
	// The targets this order is effecting, keyed by subject internal id for O(1) removal
	subjects   map[uint64]Orderable
	order_type defs.OrderType
	// What the order is targeting (could be a space, a unit, or a technology)
	target_id             int64
	clear_previous_orders bool
	subject_completed     bool
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

func (p *RisqPlayer) orderFromFrontend(o defs.OrderFromFrontend) *RisqOrder {
	order_type := defs.OrderType(o.Order_type)
	subjects := make(map[uint64]Orderable)
	for _, subject_id := range o.Subjects {
		unit, building := p.units[subject_id], p.buildings[subject_id]
		if order_type.IsUnitOrder() || (order_type == defs.OrderType_CancelOrder && unit != nil) {
			subjects[subject_id] = unit
		} else if order_type.IsBuildingOrder() || (order_type == defs.OrderType_CancelOrder && building != nil) {
			subjects[subject_id] = building
		}
	}
	return createRisqOrder(0, order_type, p.player.Player_id, subjects, o.Target_id, o.Clear_previous_orders)
}

// Drops a resolved subject; the last one marks the order executed if any subject completed, else cancelled
func (o *RisqOrder) resolveSubject(subject_id uint64, completed bool, turn uint16) {
	o.subject_completed = o.subject_completed || completed
	if len(o.subjects) > 1 {
		delete(o.subjects, subject_id)
		return
	}
	o.executed = o.subject_completed
	o.cancelled = !o.subject_completed
	o.turn_resolved = turn
}

func (o *RisqOrder) toFrontend() gin.H {
	order := gin.H{
		"internal_id":           o.internal_id,
		"player_id":             o.player_id,
		"order_type":            o.order_type,
		"target_id":             o.target_id,
		"clear_previous_orders": o.clear_previous_orders,
	}
	subjects := make([]uint64, 0, len(o.subjects))
	for _, id := range slices.Sorted(maps.Keys(o.subjects)) {
		if subject := o.subjects[id]; subject != nil && !subject.isDeleted() {
			subjects = append(subjects, id)
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
	}
}

func (q *RisqOrderQueue) receiveOrder(o *RisqOrder, prepend bool) {
	if prepend {
		q.active_orders = append([]*RisqOrder{o}, q.active_orders...)
	} else {
		q.active_orders = append(q.active_orders, o)
	}
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
		status := orderable.orderStatus(o, risq)
		if status != OrderStatus_InProgress {
			risq.recordOrderDecision(orderable, o, status)
		}
		switch status {
		case OrderStatus_InProgress:
			return o
		case OrderStatus_Cancelled:
			o.resolveSubject(orderable.internalId(), false, risq.turn_number)
		default:
			o.resolveSubject(orderable.internalId(), true, risq.turn_number)
		}
		q.active_orders = q.active_orders[1:]
	}
	return nil
}
