package risq

import (
	"fmt"
	"os"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

type RisqUnit struct {
	deleted         bool
	internal_id     uint64
	player_id       int
	unit_id         uint32
	display_name    string
	zone            *RisqZone
	turn_stamina    int
	current_stamina int
	max_stamina     int
	cs              RisqCombatStats
	order_queue     RisqOrderQueue
	intent          *RisqUnitIntent
}

func createRisqUnit(internal_id uint64, unit_id uint32, player_id int) *RisqUnit {
	unit := RisqUnit{
		deleted:         false,
		internal_id:     internal_id,
		player_id:       player_id,
		unit_id:         unit_id,
		turn_stamina:    10,
		current_stamina: 0,
		max_stamina:     15,
		cs:              createRisqCombatStats(),
		order_queue:     createRisqOrderQueue(),
		intent:          createRisqUnitIntent(),
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

func (u *RisqUnit) refreshStamina() {
	u.current_stamina += u.turn_stamina
	if u.current_stamina > u.max_stamina {
		u.current_stamina = u.max_stamina
	}
}

func (u *RisqUnit) receiveOrder(o *RisqOrder) {
	u.order_queue.receiveOrder(o)
}

func (u *RisqUnit) orderComplete(o *RisqOrder, risq *GameRisq) bool {
	switch o.order_type {
	case OrderType_UnitMoveSpace:
		x, y := util.InvertPair(uint(o.target_id))
		space := risq.getSpace(&game_utils.Coordinate2D{X: x, Y: y})
		return u.zone.space == space
	case OrderType_UnitMoveZone:
		// TODO: implement
		return false
	default:
		return true
	}
}

func (u *RisqUnit) tickIntent(risq *GameRisq) bool {
	u.intent.resetIntent()
	order := u.order_queue.nextOrder(u, risq)
	if order == nil {
		return false
	}
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
	if u.intent.intent_cost > u.current_stamina {
		u.intent.resetIntent()
	}
	return u.intent.hasIntent()
}

func (u *RisqUnit) tickExecute(risq *GameRisq) {
	if !u.intent.hasIntent() {
		return
	}
	move := u.intent.move
	if move != nil {
		fmt.Println("Moving unit", u.display_name, "to zone"+move.next_step.coordinate.ToString(), "in space", move.next_step.space.coordinate.ToString())
		old_zone := u.zone
		new_zone := move.next_step
		if old_zone.space == new_zone.space {
			delete(old_zone.units, u.internal_id)
			u.zone = new_zone
			new_zone.units[u.internal_id] = u
		} else {
			old_zone.space.removeUnit(u)
			new_zone.space.setUnit(&new_zone.coordinate, u)
		}
	}
	u.current_stamina -= u.intent.intent_cost
	fmt.Println("Unit in zone", u.zone.coordinate.ToString(), "of space", u.zone.space.coordinate.ToString())
}

func (u *RisqUnit) toFrontend() gin.H {
	unit := gin.H{
		"internal_id":     u.internal_id,
		"player_id":       u.player_id,
		"unit_id":         u.unit_id,
		"display_name":    u.display_name,
		"turn_stamina":    u.turn_stamina,
		"current_stamina": u.current_stamina,
		"max_stamina":     u.max_stamina,
		"combat_stats":    u.cs.toFrontend(),
	}
	if u.zone != nil {
		unit["zone_coordinate"] = u.zone.coordinate.ToFrontend()
		if u.zone.space != nil {
			unit["space_coordinate"] = u.zone.space.coordinate.ToFrontend()
		}
	}
	active_orders := make([]gin.H, 0)
	for _, order := range u.order_queue.active_orders {
		if order != nil && !order.executed {
			active_orders = append(active_orders, order.toFrontend())
		}
	}
	unit["active_orders"] = active_orders
	return unit
}
