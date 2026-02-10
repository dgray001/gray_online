package risq

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

type Orderable interface {
	toFrontend() gin.H
	isDeleted() bool
	internalId() uint64
	receiveOrder(o *RisqOrder)
	resolveOrders(risq *GameRisq)
}

type OrderFromFrontend struct {
	Player_id  int      `json:"player_id"`
	Subjects   []uint64 `json:"subjects"`
	Order_type uint8    `json:"order_type"`
	Target_id  int64    `json:"target_id"`
}

type OrderType uint8

const (
	OrderType_None OrderType = iota
	// A move only command where unit will not automatically attack, gather, etc
	OrderType_UnitMoveSpace
	OrderType_UnitMoveZone
	// These orders will first move the unit if necessary
	OrderType_UnitGather
	OrderType_UnitBuild
	OrderType_UnitRepair
	OrderType_UnitAttackSpace
	OrderType_UnitAttackZone
	OrderType_UnitAttackUnit
	OrderType_UnitAttackBuilding
	OrderType_UnitDefend
	OrderType_UnitGarrison
	// Orders to control buildings
	OrderType_BuildingCreate
	OrderType_BuildingResearch
	// Used to validate input from the frontend
	OrderType_END
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

func (ot OrderType) isUnitOrder() bool {
	return ot >= OrderType_UnitMoveSpace && ot <= OrderType_UnitGarrison
}

func (ot OrderType) isBuildingOrder() bool {
	return ot >= OrderType_BuildingCreate && ot <= OrderType_BuildingResearch
}

func (r *GameRisq) getOrdersFromPlayerAction(action gin.H) ([]OrderFromFrontend, error) {
	orders := make([]OrderFromFrontend, 0)
	bytes, err1 := json.Marshal(action["orders"])
	if err1 != nil {
		return orders, err1
	}
	err2 := json.Unmarshal(bytes, &orders)
	if err2 != nil {
		return orders, err2
	}
	for _, order := range orders {
		err3 := r.validateFrontendOrder(order)
		if err3 != nil {
			return orders, err3
		}
	}
	return orders, nil
}

func (r *GameRisq) validateFrontendOrder(order OrderFromFrontend) error {
	// Validate order type
	order_type := OrderType(order.Order_type)
	if order_type <= OrderType_None || order_type >= OrderType_END {
		return fmt.Errorf("Invalid order type: %d", order_type)
	}
	// Validate player id
	if order.Player_id < 0 || order.Player_id >= len(r.players) {
		return fmt.Errorf("Invalid player id: %d", order.Player_id)
	}
	// Validate order type and subjects
	if order_type.isUnitOrder() {
		for _, subject_id := range order.Subjects {
			if r.players[order.Player_id].units[subject_id] == nil {
				return fmt.Errorf("Invalid unit subject id")
			}
		}
	} else if order_type.isBuildingOrder() {
		for _, subject_id := range order.Subjects {
			if r.players[order.Player_id].buildings[subject_id] == nil {
				return fmt.Errorf("Invalid building subject id")
			}
		}
	} else {
		return errors.New("Invalid order type")
	}
	// Validate target id
	switch order_type {
	case OrderType_UnitMoveSpace:
		x, y := util.InvertPair(uint(order.Target_id))
		space := r.getSpace(&game_utils.Coordinate2D{X: x, Y: y})
		if space == nil {
			return fmt.Errorf("Invalid space target %d inverted to (%d, %d)", order.Target_id, x, y)
		}
	case OrderType_UnitMoveZone:
	default:
		return fmt.Errorf("Unimplemented order type: %d", order_type)
	}
	// Order is valid
	return nil
}
