package risq

import (
	"fmt"
	"os"

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
	intent          *RisqIntent
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
		intent:          createRisqIntent(),
	}
	config, ok := unitConfigs[unit_id]
	if !ok {
		fmt.Fprintln(os.Stderr, "Creating unknown unit id: ", unit_id)
		return &unit
	}
	unit.display_name = config.display_name
	unit.cs.setMaxHealth(config.max_health)
	unit.cs.attack_type = config.attack_type
	unit.cs.attack_blunt = config.attack_blunt
	unit.cs.attack_piercing = config.attack_piercing
	return &unit
}

func unitProductionCost(unit_id uint32) (RisqResourceCost, int) {
	config, ok := unitConfigs[unit_id]
	if !ok {
		fmt.Fprintln(os.Stderr, "Unknown unit id for production cost: ", unit_id)
		return RisqResourceCost{}, 0
	}
	return config.cost, config.production_stamina
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

func (u *RisqUnit) receiveOrder(o *RisqOrder, risq *GameRisq) {
	if o.order_type == OrderType_UnitCancelOrder {
		u.order_queue.cancelOrder(uint64(o.target_id))
		return
	}
	already_received := o.received
	u.order_queue.receiveOrder(o)
	if already_received {
		return
	}
	switch o.order_type {
	case OrderType_UnitBuild:
		building_id, _, zone := invertBuildKey(uint(o.target_id), risq)
		player := risq.players[u.player_id]
		cost, _ := buildingProductionCost(building_id)
		if !player.resources.canAfford(cost) {
			// TODO: surface this failure in the per-player turn report
			return
		}
		player.planned_foundations[zone.coordinate_key] = createRisqPlannedFoundation(building_id, o, player)
	}
}

func (u *RisqUnit) orderReceivable(o *RisqOrder, risq *GameRisq) bool {
	switch o.order_type {
	case OrderType_UnitBuild:
		_, _, zone := invertBuildKey(uint(o.target_id), risq)
		foundation := risq.players[u.player_id].planned_foundations[zone.coordinate_key]
		if zone.building != nil || (foundation != nil && foundation.creating_order != o) {
			return false
		}
	default:
	}
	return true
}

func (u *RisqUnit) orderStatus(o *RisqOrder, risq *GameRisq) OrderStatus {
	switch o.order_type {
	case OrderType_UnitMoveSpace:
		space := invertSpaceKey(uint(o.target_id), risq)
		if u.zone.space != space {
			return OrderStatus_InProgress
		}
	case OrderType_UnitMoveZone:
		_, zone := invertZoneKey(uint(o.target_id), risq)
		if u.zone != zone {
			return OrderStatus_InProgress
		}
	case OrderType_UnitGather:
		_, zone := invertZoneKey(uint(o.target_id), risq)
		if zone.resource != nil && zone.resource.resources_left > 0 {
			return OrderStatus_InProgress
		}
	case OrderType_UnitBuild:
		_, _, zone := invertBuildKey(uint(o.target_id), risq)
		if zone.building == nil {
			if risq.players[u.player_id].planned_foundations[zone.coordinate_key] == nil {
				return OrderStatus_Cancelled
			}
			return OrderStatus_InProgress
		}
		if zone.building.player_id != u.player_id {
			risq.players[u.player_id].cancelPlannedFoundation(zone)
			return OrderStatus_Cancelled
		}
		if zone.building.underConstruction() {
			return OrderStatus_InProgress
		}
	}
	return OrderStatus_Executed
}

func (u *RisqUnit) tickIntent(risq *GameRisq) bool {
	u.intent.resetIntent()
	order := u.order_queue.nextOrder(u, risq)
	if order == nil {
		return false
	}
	switch order.order_type {
	case OrderType_UnitMoveSpace:
		space := invertSpaceKey(uint(order.target_id), risq)
		u.intent.setMove(u.findPath(space.getCenterZone()))
	case OrderType_UnitMoveZone:
		_, zone := invertZoneKey(uint(order.target_id), risq)
		u.intent.setMove(u.findPath(zone))
	case OrderType_UnitGather:
		_, zone := invertZoneKey(uint(order.target_id), risq)
		if u.zone != zone {
			u.intent.setMove(u.findPath(zone))
		} else {
			u.intent.setGather(zone.resource)
		}
	case OrderType_UnitBuild:
		building_id, _, zone := invertBuildKey(uint(order.target_id), risq)
		if u.zone != zone {
			u.intent.setMove(u.findPath(zone))
		} else {
			u.intent.setBuild(zone.building, building_id, zone)
		}
	default:
		fmt.Fprintln(os.Stderr, "Order type not implemented:", order.order_type)
	}
	u.intent.resolveCost(u.current_stamina)
	return u.intent.hasIntent()
}

func (u *RisqUnit) tickExecute(risq *GameRisq) {
	if !u.intent.hasIntent() {
		return
	}
	switch detail := u.intent.detail.(type) {
	case *MoveIntent:
		fmt.Println("Moving unit", u.display_name, "to zone"+detail.next_step.coordinate.ToString(), "in space", detail.next_step.space.coordinate.ToString())
		old_zone := u.zone
		new_zone := detail.next_step
		if old_zone.space == new_zone.space {
			delete(old_zone.units, u.internal_id)
			u.zone = new_zone
			new_zone.units[u.internal_id] = u
		} else {
			old_zone.space.removeUnit(u)
			new_zone.space.setUnit(&new_zone.coordinate, u)
		}
	case *GatherIntent:
		amount := float64(u.intent.intent_cost) * (float64(detail.resource.base_gather_speed) / gatherRateStaminaBase)
		if amount > detail.resource.resources_left {
			amount = detail.resource.resources_left
		}
		detail.resource.resources_left -= amount
		risq.players[u.player_id].resources.addGathered(detail.resource.category(), amount)
	case *ConstructionIntent:
		if detail.zone.building != nil && detail.zone.building.player_id != u.player_id {
			return
		}
		building := detail.building_under_construction
		if building == nil {
			if detail.zone.building != nil {
				building = detail.zone.building
			} else {
				_, stamina_required := buildingProductionCost(detail.building_id)
				building = createRisqBuilding(risq.nextBuildingInternalId(), detail.building_id, u.player_id)
				building.stamina_remaining = stamina_required
				detail.zone.space.setBuilding(&detail.zone.coordinate, building)
				risq.players[u.player_id].buildings[building.internal_id] = building
			}
		}
		building.stamina_remaining -= u.intent.intent_cost
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
	builds := make([]gin.H, 0)
	for _, p := range unitConfigs[u.unit_id].builds {
		builds = append(builds, p.toFrontend())
	}
	unit["builds"] = builds
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
