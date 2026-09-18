package risq

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
)

type OrderableType uint8

const (
	OrderableType_NONE OrderableType = iota
	OrderableType_BUILDING
	OrderableType_UNIT
)

type Orderable interface {
	toFrontend(viewer_player_id int) gin.H
	isDeleted() bool
	internalId() uint64
	OrderableType() OrderableType
	activeOrders() []*RisqOrder
	refreshStamina()
	// Returns whether the order is receivable by this subject
	orderReceivable(o *RisqOrder, risq *GameRisq) bool
	// Returns failure reason if order was not received
	receiveOrder(o *RisqOrder, risq *GameRisq) error
	cancelOrder(o *RisqOrder, risq *GameRisq)
	// Returns whether the order is in progress, executed, or cancelled (called by tickIntent)
	orderStatus(o *RisqOrder, risq *GameRisq) OrderStatus
	// Returns whether the orderable has an intent
	tickIntent(risq *GameRisq) bool
	tickExecute(risq *GameRisq)
	// Removes this orderable from all maps/zones/orders once deleted; called once per player per turn
	cleanupDeleted(risq *GameRisq)
}

// Resolves orders naming internal_id as a subject; shared by RisqUnit/RisqBuilding's cleanupDeleted
func resolveOrdersOnDeath(risq *GameRisq, queue *RisqOrderQueue, internal_id uint64, self_delete_order_type OrderType) {
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

type OrderFromFrontend struct {
	Player_id             int      `json:"player_id"`
	Subjects              []uint64 `json:"subjects"`
	Order_type            uint8    `json:"order_type"`
	Target_id             int64    `json:"target_id"`
	Clear_previous_orders bool     `json:"clear_previous_orders"`
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
	// Server synthesized; not submitted by player
	OrderType_UnitAutoAttackUnit
	OrderType_UnitDefend
	OrderType_UnitGarrison
	OrderType_UnitUngarrison
	OrderType_UnitDelete
	// Orders to control buildings
	OrderType_BuildingCreate
	OrderType_BuildingResearch
	OrderType_BuildingDelete
	// Player-level orders with no subjects
	OrderType_CancelOrder
	OrderType_CancelFoundation
	// Used to validate input from the frontend
	OrderType_END
)

type RisqOrder struct {
	// Internal id of the order object itself
	internal_id uint64
	// The player id of who is creating this order
	player_id int
	// The targets this order is effecting, keyed by subject internal id for O(1) removal
	subjects map[uint64]Orderable
	// What the order actually is
	order_type OrderType
	// What the order is targeting (could be a space, a unit, or a technology)
	target_id int64
	// Whether receiving this order should cancel each subject's other active orders
	clear_previous_orders bool
	// Whether this order has been received (used for one-time effects)
	received bool
	// Whether the order has been executed
	executed bool
	// Whether the order was canceled before completion
	cancelled bool
	// The turn that the order was received by the player
	turn_received uint16
	// The turn that the order was resolved (executed or cancelled)
	turn_resolved uint16
}

func createRisqOrder(internal_id uint64, order_type OrderType, player_id int, subjects map[uint64]Orderable, target_id int64, clear_previous_orders bool) *RisqOrder {
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
		"internal_id":   o.internal_id,
		"player_id":     o.player_id,
		"order_type":    o.order_type,
		"target_id":     o.target_id,
		"turn_received": o.turn_received,
		"turn_resolved": o.turn_resolved,
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
	return ot >= OrderType_UnitMoveSpace && ot <= OrderType_UnitDelete
}

func (ot OrderType) isBuildingOrder() bool {
	return ot >= OrderType_BuildingCreate && ot <= OrderType_BuildingDelete
}

func (ot OrderType) isPlayerOrder() bool {
	return ot >= OrderType_CancelOrder && ot <= OrderType_CancelFoundation
}

type UnitBehaviorFromFrontend struct {
	Internal_ids      []uint64 `json:"internal_ids"`
	Stance            *uint8   `json:"stance"`
	Interrupt_current *bool    `json:"interrupt_current"`
	Attack_back       *bool    `json:"attack_back"`
	Target_priority   *[]uint8 `json:"target_priority"`
}

func getUnitBehaviorFromPlayerAction(action gin.H) (UnitBehaviorFromFrontend, error) {
	var behavior UnitBehaviorFromFrontend
	bytes, err := json.Marshal(action)
	if err != nil {
		return behavior, err
	}
	if err := json.Unmarshal(bytes, &behavior); err != nil {
		return behavior, err
	}
	return behavior, nil
}

type GatherPointFromFrontend struct {
	Building_id   uint64 `json:"building_id"`
	Clear         bool   `json:"clear"`
	Location_kind uint8  `json:"location_kind"`
	Location_id   uint64 `json:"location_id"`
	Object_type   uint8  `json:"object_type"`
	Object_id     uint64 `json:"object_id"`
}

func getGatherPointFromPlayerAction(action gin.H) (GatherPointFromFrontend, error) {
	var request GatherPointFromFrontend
	bytes, err := json.Marshal(action)
	if err != nil {
		return request, err
	}
	if err := json.Unmarshal(bytes, &request); err != nil {
		return request, err
	}
	return request, nil
}

func (r *GameRisq) getOrdersFromPlayerAction(action gin.H, player_id int) ([]OrderFromFrontend, error) {
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
		err3 := r.validateFrontendOrder(order, player_id)
		if err3 != nil {
			return orders, err3
		}
	}
	return orders, nil
}

func (r *GameRisq) validateFrontendOrder(order OrderFromFrontend, player_id int) error {
	// Validate order type
	order_type := OrderType(order.Order_type)
	if order_type <= OrderType_None || order_type >= OrderType_END || order_type == OrderType_UnitAutoAttackUnit {
		return fmt.Errorf("Invalid order type: %d", order_type)
	}
	// Order owner must be the submitting player
	if order.Player_id != player_id {
		return fmt.Errorf("Order player id %d does not match submitter %d", order.Player_id, player_id)
	}
	// Validate order type and subjects
	if order_type.isUnitOrder() {
		if len(order.Subjects) == 0 {
			return errors.New("Order must have at least one subject")
		}
		for _, subject_id := range order.Subjects {
			if r.players[order.Player_id].units[subject_id] == nil {
				return fmt.Errorf("Invalid unit subject id")
			}
		}
	} else if order_type.isBuildingOrder() {
		if len(order.Subjects) == 0 {
			return errors.New("Order must have at least one subject")
		}
		for _, subject_id := range order.Subjects {
			if r.players[order.Player_id].buildings[subject_id] == nil {
				return fmt.Errorf("Invalid building subject id")
			}
		}
	} else if order_type.isPlayerOrder() {
		if len(order.Subjects) > 0 {
			return errors.New("Invalid subjects in player order")
		}
	}
	// Validate target id
	switch order_type {
	case OrderType_UnitMoveSpace:
		space := invertSpaceKey(uint(order.Target_id), r)
		if space == nil {
			return fmt.Errorf("Invalid space target inverted from %d", order.Target_id)
		}
	case OrderType_UnitMoveZone:
		space, zone := invertZoneKey(uint(order.Target_id), r)
		if space == nil {
			return fmt.Errorf("Invalid space target inverted from zone key %d", order.Target_id)
		}
		if zone == nil {
			return fmt.Errorf("Invalid zone target inverted from zone key %d", order.Target_id)
		}
	case OrderType_UnitGather:
		space, zone := invertZoneKey(uint(order.Target_id), r)
		if space == nil {
			return fmt.Errorf("Invalid space target inverted from zone key %d", order.Target_id)
		}
		if zone == nil {
			return fmt.Errorf("Invalid zone target inverted from zone key %d", order.Target_id)
		}
		if zone.resource == nil && (zone.building == nil || !buildingConfigs[zone.building.building_id].isGatherable()) {
			return fmt.Errorf("No resource in target zone")
		}
		for _, subject_id := range order.Subjects {
			if r.players[order.Player_id].units[subject_id].unitType() != UnitType_ECONOMIC {
				return fmt.Errorf("Only economic units can gather")
			}
		}
	case OrderType_BuildingCreate:
		unit_id := uint32(order.Target_id)
		unit_config, ok := unitConfigs[unit_id]
		if !ok {
			return fmt.Errorf("Invalid or unsupported unit id for production: %d", unit_id)
		}
		if !requiredTechMet(r.players[order.Player_id], unit_config.required_tech_id) {
			return fmt.Errorf("Required tech id %d not researched for unit id %d", unit_config.required_tech_id, unit_id)
		}
		for _, subject_id := range order.Subjects {
			building := r.players[order.Player_id].buildings[subject_id]
			if building.underConstruction() {
				return fmt.Errorf("Building id %d is still under construction", subject_id)
			}
			if !buildingConfigs[building.building_id].canProduce(unit_id) {
				return fmt.Errorf("Building id %d cannot produce unit id %d", building.building_id, unit_id)
			}
		}
	case OrderType_BuildingResearch:
		tech_id := uint32(order.Target_id)
		tech_config, ok := techConfigs[tech_id]
		if !ok {
			return fmt.Errorf("Invalid or unsupported tech id: %d", tech_id)
		}
		if !requiredTechMet(r.players[order.Player_id], tech_config.required_tech_id) {
			return fmt.Errorf("Required tech id %d not researched for tech id %d", tech_config.required_tech_id, tech_id)
		}
		if researched, exists := r.players[order.Player_id].researched_techs[tech_id]; exists {
			if researched {
				return fmt.Errorf("Tech id %d is already researched", tech_id)
			}
			return fmt.Errorf("Tech id %d is already being researched", tech_id)
		}
		for _, subject_id := range order.Subjects {
			building := r.players[order.Player_id].buildings[subject_id]
			if building.underConstruction() {
				return fmt.Errorf("Building id %d is still under construction", subject_id)
			}
			if !buildingConfigs[building.building_id].canResearch(tech_id) {
				return fmt.Errorf("Building id %d cannot research tech id %d", building.building_id, tech_id)
			}
		}
	case OrderType_UnitBuild:
		building_id, space, zone := invertBuildKey(uint(order.Target_id), r)
		if space == nil || zone == nil {
			return fmt.Errorf("Invalid space or zone target inverted from build key %d", order.Target_id)
		}
		_, stamina_required := buildingProductionCost(building_id)
		if stamina_required <= 0 {
			return fmt.Errorf("Invalid or unbuildable building id: %d", building_id)
		}
		if !requiredTechMet(r.players[order.Player_id], buildingConfigs[building_id].required_tech_id) {
			return fmt.Errorf("Required tech id %d not researched for building id %d", buildingConfigs[building_id].required_tech_id, building_id)
		}
		if zone.resource != nil {
			return fmt.Errorf("Target zone is already occupied")
		}
		if zone.building != nil {
			b := zone.building
			if b.player_id != order.Player_id || !b.underConstruction() || b.building_id != building_id {
				return fmt.Errorf("Target zone is already occupied")
			}
		}
		for _, subject_id := range order.Subjects {
			unit := r.players[order.Player_id].units[subject_id]
			if unit.unitType() != UnitType_ECONOMIC {
				return fmt.Errorf("Only economic units can build")
			}
			if !unitConfigs[unit.unit_id].canBuild(building_id) {
				return fmt.Errorf("Unit id %d cannot build building id %d", unit.unit_id, building_id)
			}
		}
	case OrderType_UnitRepair:
		building := r.buildings[uint64(order.Target_id)]
		if building == nil {
			return fmt.Errorf("Invalid building target id %d", order.Target_id)
		}
		if building.underConstruction() {
			return fmt.Errorf("Cannot repair a building under construction")
		}
		for _, subject_id := range order.Subjects {
			if r.players[order.Player_id].units[subject_id].unitType() != UnitType_ECONOMIC {
				return fmt.Errorf("Only economic units can repair")
			}
		}
	case OrderType_UnitAttackUnit:
		if r.units[uint64(order.Target_id)] == nil {
			return fmt.Errorf("Invalid unit target id %d", order.Target_id)
		}
	case OrderType_UnitAttackBuilding:
		if r.buildings[uint64(order.Target_id)] == nil {
			return fmt.Errorf("Invalid building target id %d", order.Target_id)
		}
	case OrderType_UnitAttackZone:
		space, zone := invertZoneKey(uint(order.Target_id), r)
		if space == nil {
			return fmt.Errorf("Invalid space target inverted from zone key %d", order.Target_id)
		}
		if zone == nil {
			return fmt.Errorf("Invalid zone target inverted from zone key %d", order.Target_id)
		}
	case OrderType_UnitAttackSpace:
		space := invertSpaceKey(uint(order.Target_id), r)
		if space == nil {
			return fmt.Errorf("Invalid space target inverted from %d", order.Target_id)
		}
	case OrderType_UnitGarrison:
		if r.buildings[uint64(order.Target_id)] == nil {
			return fmt.Errorf("Invalid building target id %d", order.Target_id)
		}
	case OrderType_UnitUngarrison:
	case OrderType_UnitDelete:
	case OrderType_BuildingDelete:
	case OrderType_CancelOrder:
		found := false
		for _, active_order := range r.players[order.Player_id].active_orders {
			if active_order.internal_id == uint64(order.Target_id) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("No active order with id %d", order.Target_id)
		}
	case OrderType_CancelFoundation:
		_, zone := invertZoneKey(uint(order.Target_id), r)
		if zone == nil {
			return fmt.Errorf("Invalid zone target inverted from zone key %d", order.Target_id)
		}
		if r.players[order.Player_id].planned_foundations[zone.coordinate_key] == nil {
			return fmt.Errorf("No planned foundation at this zone")
		}
	default:
		return fmt.Errorf("Unimplemented order type: %d", order_type)
	}
	// Order is valid
	return nil
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

// Finds, cancels, and removes the order with this internal id; returns it (or nil if not found)
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
