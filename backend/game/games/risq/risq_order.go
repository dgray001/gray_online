package risq

import "github.com/gin-gonic/gin"

type Orderable interface {
	toFrontend() gin.H
	isDeleted() bool
	internalId() uint64
	receiveOrder(o *RisqOrder) bool
	resolveOrders(risq *GameRisq)
}

type OrderType uint8

const (
	OrderType_None OrderType = iota
	// A move only command where unit will not automatically attack, gather, etc
	OrderType_UnitMove
	// These orders will first move the unit if necessary
	OrderType_UnitGather
	OrderType_UnitBuild
	OrderType_UnitRepair
	OrderType_UnitAttack
	OrderType_UnitDefend
	OrderType_UnitGarrison
	// Orders to control buildings
	OrderType_BuildingCreate
	OrderType_BuildingResearch
)

type RisqOrder struct {
	// Internal id of the order object itself
	internal_id uint64
	// The player id of who is creating this order
	player_id int
	// The list of targets this order is effecting
	subjects []Orderable
	// What the order actually is
	order_type OrderType
	// What the order is targeting (could be a space, a unit, or a technology)
	target_id int64
	// Whether the order has been executed
	executed bool
	// The turn that the order was received by the player
	turn_received uint16
	// The turn that the order was executed
	turn_executed uint16
}

func createRisqOrder(internal_id uint64, order_type OrderType, player_id int, subjects []Orderable, target_id int64) *RisqOrder {
	order := RisqOrder{
		internal_id: internal_id,
		player_id:   player_id,
		subjects:    subjects,
		order_type:  order_type,
		target_id:   target_id,
	}
	return &order
}

func (o *RisqOrder) toFrontend() gin.H {
	order := gin.H{
		"internal_id":   o.internal_id,
		"player_id":     o.player_id,
		"order_type":    o.order_type,
		"target_id":     o.target_id,
		"turn_received": o.turn_received,
		"turn_executed": o.turn_executed,
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
