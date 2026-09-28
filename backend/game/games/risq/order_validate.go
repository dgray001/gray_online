package risq

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
)

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

type BuildingBehaviorFromFrontend struct {
	Internal_ids      []uint64 `json:"internal_ids"`
	Auto_attack       *bool    `json:"auto_attack"`
	Interrupt_current *bool    `json:"interrupt_current"`
	Target_priority   *[]uint8 `json:"target_priority"`
}

func getBuildingBehaviorFromPlayerAction(action gin.H) (BuildingBehaviorFromFrontend, error) {
	var behavior BuildingBehaviorFromFrontend
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

func (r *GameRisq) getOrdersFromPlayerAction(action gin.H, player_id int) ([]defs.OrderFromFrontend, error) {
	orders := make([]defs.OrderFromFrontend, 0)
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

func (r *GameRisq) validateFrontendOrder(order defs.OrderFromFrontend, player_id int) error {
	order_type := defs.OrderType(order.Order_type)
	if order_type <= defs.OrderType_None || order_type >= defs.OrderType_END || order_type.IsAutoSynthesized() {
		return fmt.Errorf("Invalid order type: %d", order_type)
	}
	if order.Player_id != player_id {
		return fmt.Errorf("Order player id %d does not match submitter %d", order.Player_id, player_id)
	}
	if order_type.IsUnitOrder() {
		if len(order.Subjects) == 0 {
			return errors.New("Order must have at least one subject")
		}
		for _, subject_id := range order.Subjects {
			unit := r.players[order.Player_id].units[subject_id]
			if unit == nil {
				return fmt.Errorf("Invalid unit subject id")
			}
			if order_type.IsAttackOrder() && unit.cs.attack_type == defs.AttackType_NONE {
				return fmt.Errorf("Unit id %d cannot attack", subject_id)
			}
		}
	} else if order_type.IsBuildingOrder() {
		if len(order.Subjects) == 0 {
			return errors.New("Order must have at least one subject")
		}
		for _, subject_id := range order.Subjects {
			building := r.players[order.Player_id].buildings[subject_id]
			if building == nil {
				return fmt.Errorf("Invalid building subject id")
			}
			if order_type.IsAttackOrder() && building.cs.attack_type == defs.AttackType_NONE {
				return fmt.Errorf("Building id %d cannot attack", subject_id)
			}
		}
	} else if order_type.IsPlayerOrder() {
		if len(order.Subjects) > 0 {
			return errors.New("Invalid subjects in player order")
		}
	}
	switch order_type {
	case defs.OrderType_UnitMoveSpace:
		space := invertSpaceKey(uint(order.Target_id), r)
		if space == nil {
			return fmt.Errorf("Invalid space target inverted from %d", order.Target_id)
		}
	case defs.OrderType_UnitMoveZone:
		space, zone := invertZoneKey(uint(order.Target_id), r)
		if space == nil {
			return fmt.Errorf("Invalid space target inverted from zone key %d", order.Target_id)
		}
		if zone == nil {
			return fmt.Errorf("Invalid zone target inverted from zone key %d", order.Target_id)
		}
	case defs.OrderType_UnitGather:
		space, zone := invertZoneKey(uint(order.Target_id), r)
		if space == nil {
			return fmt.Errorf("Invalid space target inverted from zone key %d", order.Target_id)
		}
		if zone == nil {
			return fmt.Errorf("Invalid zone target inverted from zone key %d", order.Target_id)
		}
		_, known_resource := zone.resourceKnownTo(order.Player_id)
		known_building, has_building := zone.buildingKnownTo(order.Player_id)
		if !known_resource && (!has_building || !defs.BuildingConfigs[known_building.building_id].IsGatherable()) {
			return fmt.Errorf("No resource in target zone")
		}
		for _, subject_id := range order.Subjects {
			if r.players[order.Player_id].units[subject_id].unitType() != defs.UnitType_ECONOMIC {
				return fmt.Errorf("Only economic units can gather")
			}
		}
	case defs.OrderType_BuildingCreate:
		unit_id := uint32(order.Target_id)
		unit_config, ok := defs.UnitConfigs[unit_id]
		if !ok {
			return fmt.Errorf("Invalid or unsupported unit id for production: %d", unit_id)
		}
		if !requiredTechMet(r.players[order.Player_id], unit_config.Required_tech_id) {
			return fmt.Errorf("Required tech id %d not researched for unit id %d", unit_config.Required_tech_id, unit_id)
		}
		for _, subject_id := range order.Subjects {
			building := r.players[order.Player_id].buildings[subject_id]
			if building.underConstruction() {
				return fmt.Errorf("Building id %d is still under construction", subject_id)
			}
			if !defs.BuildingConfigs[building.building_id].CanProduce(unit_id) {
				return fmt.Errorf("Building id %d cannot produce unit id %d", building.building_id, unit_id)
			}
		}
	case defs.OrderType_BuildingResearch:
		tech_id := uint32(order.Target_id)
		tech_config, ok := defs.TechConfigs[tech_id]
		if !ok {
			return fmt.Errorf("Invalid or unsupported tech id: %d", tech_id)
		}
		if !requiredTechMet(r.players[order.Player_id], tech_config.Required_tech_id) {
			return fmt.Errorf("Required tech id %d not researched for tech id %d", tech_config.Required_tech_id, tech_id)
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
			if !defs.BuildingConfigs[building.building_id].CanResearch(tech_id) {
				return fmt.Errorf("Building id %d cannot research tech id %d", building.building_id, tech_id)
			}
		}
	case defs.OrderType_UnitBuild:
		building_id, space, zone := invertBuildKey(uint(order.Target_id), r)
		if space == nil || zone == nil {
			return fmt.Errorf("Invalid space or zone target inverted from build key %d", order.Target_id)
		}
		_, stamina_required := defs.BuildingProductionCost(building_id)
		if stamina_required <= 0 {
			return fmt.Errorf("Invalid or unbuildable building id: %d", building_id)
		}
		if !requiredTechMet(r.players[order.Player_id], defs.BuildingConfigs[building_id].Required_tech_id) {
			return fmt.Errorf("Required tech id %d not researched for building id %d", defs.BuildingConfigs[building_id].Required_tech_id, building_id)
		}
		if _, known := zone.resourceKnownTo(order.Player_id); known {
			return fmt.Errorf("Target zone is already occupied")
		}
		if b, known := zone.buildingKnownTo(order.Player_id); known {
			if b.player_id != order.Player_id || !b.under_construction || b.building_id != building_id {
				return fmt.Errorf("Target zone is already occupied")
			}
		}
		for _, subject_id := range order.Subjects {
			unit := r.players[order.Player_id].units[subject_id]
			if unit.unitType() != defs.UnitType_ECONOMIC {
				return fmt.Errorf("Only economic units can build")
			}
			if !defs.UnitConfigs[unit.unit_id].CanBuild(building_id) {
				return fmt.Errorf("Unit id %d cannot build building id %d", unit.unit_id, building_id)
			}
		}
	case defs.OrderType_UnitRepair:
		building := r.buildings[uint64(order.Target_id)]
		if building == nil {
			return fmt.Errorf("Invalid building target id %d", order.Target_id)
		}
		if building.underConstruction() {
			return fmt.Errorf("Cannot repair a building under construction")
		}
		for _, subject_id := range order.Subjects {
			if r.players[order.Player_id].units[subject_id].unitType() != defs.UnitType_ECONOMIC {
				return fmt.Errorf("Only economic units can repair")
			}
		}
	case defs.OrderType_UnitRenew:
		building := r.buildings[uint64(order.Target_id)]
		if building == nil {
			return fmt.Errorf("Invalid building target id %d", order.Target_id)
		}
		if !defs.BuildingConfigs[building.building_id].IsGatherable() {
			return fmt.Errorf("Building id %d is not gatherable", building.building_id)
		}
		for _, subject_id := range order.Subjects {
			if r.players[order.Player_id].units[subject_id].unitType() != defs.UnitType_ECONOMIC {
				return fmt.Errorf("Only economic units can renew")
			}
		}
	case defs.OrderType_UnitAttackUnit:
		if r.units[uint64(order.Target_id)] == nil {
			return fmt.Errorf("Invalid unit target id %d", order.Target_id)
		}
	case defs.OrderType_UnitAttackBuilding:
		if r.buildings[uint64(order.Target_id)] == nil {
			return fmt.Errorf("Invalid building target id %d", order.Target_id)
		}
	case defs.OrderType_BuildingAttackUnit:
		if r.units[uint64(order.Target_id)] == nil {
			return fmt.Errorf("Invalid unit target id %d", order.Target_id)
		}
	case defs.OrderType_BuildingAttackBuilding:
		if r.buildings[uint64(order.Target_id)] == nil {
			return fmt.Errorf("Invalid building target id %d", order.Target_id)
		}
	case defs.OrderType_UnitAttackZone:
		space, zone := invertZoneKey(uint(order.Target_id), r)
		if space == nil {
			return fmt.Errorf("Invalid space target inverted from zone key %d", order.Target_id)
		}
		if zone == nil {
			return fmt.Errorf("Invalid zone target inverted from zone key %d", order.Target_id)
		}
	case defs.OrderType_UnitAttackSpace:
		space := invertSpaceKey(uint(order.Target_id), r)
		if space == nil {
			return fmt.Errorf("Invalid space target inverted from %d", order.Target_id)
		}
	case defs.OrderType_UnitGarrison:
		if r.buildings[uint64(order.Target_id)] == nil {
			return fmt.Errorf("Invalid building target id %d", order.Target_id)
		}
	case defs.OrderType_UnitUngarrison:
	case defs.OrderType_UnitDelete:
	case defs.OrderType_BuildingDelete:
	case defs.OrderType_CancelOrder:
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
	case defs.OrderType_CancelFoundation:
		_, zone := invertZoneKey(uint(order.Target_id), r)
		if zone == nil {
			return fmt.Errorf("Invalid zone target inverted from zone key %d", order.Target_id)
		}
		if r.players[order.Player_id].planned_foundations[zone.coordinate_key] == nil {
			return fmt.Errorf("No planned foundation at this zone")
		}
	case defs.OrderType_BuyMercenary:
		unit_id, space, zone := invertMercenaryKey(uint(order.Target_id), r)
		if space == nil || zone == nil {
			return fmt.Errorf("Invalid space or zone target inverted from mercenary key %d", order.Target_id)
		}
		if _, ok := defs.UnitConfigs[unit_id]; !ok {
			return fmt.Errorf("Invalid or unsupported unit id: %d", unit_id)
		}
		if !r.players[order.Player_id].available_mercenaries[unit_id] {
			return fmt.Errorf("Unit id %d is not an available mercenary", unit_id)
		}
	default:
		return fmt.Errorf("Unimplemented order type: %d", order_type)
	}
	return nil
}
