package risq

import (
	"sort"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
)

func buildingProducesToFrontend(config *defs.Config, building_id uint32) []gin.H {
	produces := make([]gin.H, 0)
	for _, p := range config.Buildings[building_id].Produces {
		produces = append(produces, p.ToFrontend(config))
	}
	return produces
}

func (b *RisqBuilding) toFrontend(viewer_player_id int) gin.H {
	config := b.config.Buildings[b.building_id]
	display_name := b.display_name
	if config.IsGatherable() && b.resources_left <= 0 {
		display_name += " (expired)"
	}
	building := gin.H{
		"internal_id":                b.internal_id,
		"player_id":                  b.player_id,
		"building_id":                b.building_id,
		"display_name":               display_name,
		"population_support":         b.population_support,
		"combat_stats":               b.cs.toFrontend(),
		"under_construction":         b.underConstruction(),
		"stamina_remaining":          b.stamina_remaining,
		"construction_stamina_total": b.construction_stamina_total,
		"garrison_capacity":          b.garrison_capacity,
		"attack_range":               b.attack_range,
	}
	building["has_garrisoned_units"] = len(b.garrisoned_units) > 0
	building["tick_actions"] = tickActionsToFrontend(b.tick_actions, viewer_player_id)
	if showOrdersTo(b.player_id, b.zone, viewer_player_id) {
		garrisoned_units := make([]uint64, 0)
		for id := range b.garrisoned_units {
			garrisoned_units = append(garrisoned_units, id)
		}
		building["garrisoned_units"] = garrisoned_units
	}
	if showOrdersTo(b.player_id, b.zone, viewer_player_id) {
		building["turn_stamina"] = b.turn_stamina
		building["current_stamina"] = b.current_stamina
		building["max_stamina"] = maxStaminaFor(b.turn_stamina)
		building["renewing"] = b.renewing != nil
		if b.renewing != nil {
			building["renew_stamina_remaining"] = b.renew_stamina_remaining
		}
		building["auto_attack"] = b.auto_attack
		building["interrupt_current"] = b.interrupt_current
		target_priority := make([]int, len(b.target_priority))
		for i, cat := range b.target_priority {
			target_priority[i] = int(cat)
		}
		building["target_priority"] = target_priority
	}
	if b.gather_point != nil && showOrdersTo(b.player_id, b.zone, viewer_player_id) {
		building["gather_point"] = b.gather_point.toFrontend()
	}
	building["produces"] = buildingProducesToFrontend(b.config, b.building_id)
	if config.IsGatherable() {
		building["resources_left"] = b.resources_left
		building["resource_capacity"] = config.Gather.Starting_resources
		building["renew_stamina"] = config.Gather.Renew_stamina
		building["gather_capacity"] = config.Gather.Gather_capacity
		building["base_gather_speed"] = config.Gather.Base_gather_speed
		building["renew_cost"] = config.Gather.Renew_cost.ToFrontend()
		building["resource_category"] = config.Gather.Resource_category
	}
	if b.zone != nil {
		building["zone_coordinate"] = b.zone.coordinate.ToFrontend()
		if b.zone.space != nil {
			building["space_coordinate"] = b.zone.space.coordinate.ToFrontend()
		}
	}
	active_orders := make([]gin.H, 0)
	production_queue := make([]gin.H, 0)
	if showOrdersTo(b.player_id, b.zone, viewer_player_id) {
		for _, order := range b.order_queue.active_orders {
			if order == nil || order.executed {
				continue
			}
			active_orders = append(active_orders, order.toFrontend())
			if item, ok := b.production_queue[order.internal_id]; ok {
				item_frontend := item.toFrontend()
				item_frontend["order_internal_id"] = order.internal_id
				production_queue = append(production_queue, item_frontend)
			}
		}
	}
	building["active_orders"] = active_orders
	building["production_queue"] = production_queue
	return building
}

func AllBuildingConfigsToFrontend(game_config *defs.Config) []gin.H {
	ids := make([]uint32, 0, len(game_config.Buildings))
	for id := range game_config.Buildings {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	buildings := make([]gin.H, len(ids))
	for i, id := range ids {
		config := game_config.Buildings[id]
		produces := make([]gin.H, len(config.Produces))
		for j, p := range config.Produces {
			produces[j] = p.ToFrontend(game_config)
		}
		buildings[i] = gin.H{
			"building_id":        id,
			"display_name":       config.Display_name,
			"description":        config.Description,
			"required_tech_id":   config.Required_tech_id,
			"buildable":          game_config.AnyUnitCanBuild(id),
			"produces":           produces,
			"cost":               config.Cost.ToFrontend(),
			"stamina_cost":       config.Build_stamina,
			"population_support": config.Population_support,
			"garrison_capacity":  config.Garrison_capacity,
			"is_wall":            config.Is_wall,
			"stats": gin.H{
				"health":               config.Max_health,
				"attack_type":          config.Attack_type,
				"attack_blunt":         config.Attack_blunt,
				"attack_piercing":      config.Attack_piercing,
				"attack_range":         config.Attack_range,
				"defense_blunt":        config.Defense_blunt,
				"defense_piercing":     config.Defense_piercing,
				"penetration_blunt":    config.Penetration_blunt,
				"penetration_piercing": config.Penetration_piercing,
			},
		}
	}
	return buildings
}
