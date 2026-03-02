package risq

import (
	"fmt"
	"os"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

type RisqUnit struct {
	deleted       bool
	internal_id   uint64
	player_id     int
	unit_id       uint32
	display_name  string
	zone          *RisqZone
	stamina       int
	cs            RisqCombatStats
	active_orders []*RisqOrder
	past_orders   []*RisqOrder
	intent        *RisqUnitIntent
}

func createRisqUnit(internal_id uint64, unit_id uint32, player_id int) *RisqUnit {
	unit := RisqUnit{
		deleted:       false,
		internal_id:   internal_id,
		player_id:     player_id,
		unit_id:       unit_id,
		stamina:       5,
		cs:            createRisqCombatStats(),
		active_orders: make([]*RisqOrder, 0),
		past_orders:   make([]*RisqOrder, 0),
		intent:        createRisqUnitIntent(),
	}
	switch unit_id {
	case 1: // villager
		unit.display_name = "Villager"
		unit.cs.setMaxHealth(25)
		unit.cs.attack_type = AttackType_BLUNT
		unit.cs.attack_blunt = 3
	case 11: // swordsman
		unit.display_name = "Swordsman"
		unit.cs.setMaxHealth(35)
		unit.cs.attack_type = AttackType_BLUNT_PIERCING
		unit.cs.attack_blunt = 4
		unit.cs.attack_piercing = 4
	default:
		fmt.Fprintln(os.Stderr, "Creating unknown unit id: ", unit_id)
	}
	return &unit
}

func (u *RisqUnit) vision() *RisqVision {
	return &RisqVision{
		space:         4,
		edge_adjacent: 4,
		adjacent:      3,
		edge_opposite: 2,
		secondary:     0,
	}
}

func (u *RisqUnit) score() uint {
	return 0
}

func (u *RisqUnit) isDeleted() bool {
	return u.deleted
}

func (u *RisqUnit) internalId() uint64 {
	return u.internal_id
}

func (u *RisqUnit) receiveOrder(o *RisqOrder) {
	u.past_orders = append(u.past_orders, o)
	u.active_orders = append(u.active_orders, o)
}

func (u *RisqUnit) tickIntent(risq *GameRisq) bool {
	u.intent.resetIntent()
	if len(u.active_orders) == 0 {
		return false
	}
	order := u.active_orders[0]
	switch order.order_type {
	case OrderType_UnitMoveSpace:
		x, y := util.InvertPair(uint(order.target_id))
		space := risq.getSpace(&game_utils.Coordinate2D{X: x, Y: y})
		u.intent.setMove(u.findPath(space.getCenterZone()))
	case OrderType_UnitMoveZone:
		// TODO: implement
	default:
		fmt.Fprintln(os.Stderr, "Order type not implemented:", order.order_type)
	}
	return u.intent.hasIntent()
}

func (u *RisqUnit) tickExecute(risq *GameRisq) {
	if !u.intent.hasIntent() {
		return
	}
	// TODO: implement
}

func (u *RisqUnit) toFrontend() gin.H {
	unit := gin.H{
		"internal_id":  u.internal_id,
		"player_id":    u.player_id,
		"unit_id":      u.unit_id,
		"display_name": u.display_name,
		"stamina":      u.stamina,
		"combat_stats": u.cs.toFrontend(),
	}
	if u.zone != nil {
		unit["zone_coordinate"] = u.zone.coordinate.ToFrontend()
		if u.zone.space != nil {
			unit["space_coordinate"] = u.zone.space.coordinate.ToFrontend()
		}
	}
	active_orders := make([]gin.H, 0)
	for _, order := range u.active_orders {
		if order != nil && !order.executed {
			active_orders = append(active_orders, order.toFrontend())
		}
	}
	unit["active_orders"] = active_orders
	return unit
}
