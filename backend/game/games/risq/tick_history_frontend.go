package risq

import (
	"github.com/gin-gonic/gin"
	"maps"
)

func (s tickUnitState) toFrontend() gin.H {
	result := gin.H{
		"internal_id": s.target.internal_id, "player_id": s.target.player_id,
		"unit_id": s.unit_id, "unit_type": s.unit_type, "display_name": s.display_name,
		"space_coordinate": s.location.space.ToFrontend(), "zone_coordinate": s.location.zone.ToFrontend(),
		"combat_stats": s.cs.toFrontend(), "current_stamina": s.stamina,
		"turn_stamina": s.turn_stamina, "max_stamina": maxStaminaFor(s.turn_stamina), "deleted": s.deleted,
	}
	if s.location.garrisoned_in != 0 {
		result["garrisoned_in"] = s.location.garrisoned_in
	}
	return result
}

func (s tickBuildingState) toFrontend() gin.H {
	return gin.H{
		"internal_id": s.target.internal_id, "player_id": s.target.player_id,
		"building_id": s.building_id, "display_name": s.display_name,
		"space_coordinate": s.location.space.ToFrontend(), "zone_coordinate": s.location.zone.ToFrontend(),
		"combat_stats": s.cs.toFrontend(), "current_stamina": s.stamina, "turn_stamina": s.turn_stamina,
		"under_construction": s.under_construction, "stamina_remaining": s.stamina_remaining,
		"construction_stamina_total": s.construction_total, "resources_left": s.resources_left,
		"renew_stamina_remaining": s.renew_stamina_remaining, "deleted": s.deleted,
	}
}

func (s tickResourceState) toFrontend() gin.H {
	return gin.H{
		"internal_id": s.target.internal_id, "resource_id": s.resource_id,
		"space_coordinate": s.location.space.ToFrontend(), "zone_coordinate": s.location.zone.ToFrontend(),
		"resources_left": s.remaining, "deleted": s.deleted,
	}
}

func (f tickFrame) toFrontend(player_id int) gin.H {
	units, buildings, resources, players, spaces, effects := []gin.H{}, []gin.H{}, []gin.H{}, []gin.H{}, []gin.H{}, []gin.H{}
	for _, state := range f.units {
		if state.visibility.see(player_id, state.target.player_id) {
			units = append(units, state.toFrontend())
		}
	}
	for _, state := range f.buildings {
		if state.visibility.see(player_id, state.target.player_id) {
			buildings = append(buildings, state.toFrontend())
		}
	}
	for _, state := range f.resources {
		if state.visibility.see(player_id, -2) {
			resources = append(resources, state.toFrontend())
		}
	}
	for _, state := range f.players {
		if player_id == state.player_id {
			players = append(players, gin.H{"player_id": state.player_id, "resources": state.resources.ToFrontend(), "researched_techs": maps.Clone(state.techs), "eliminated": state.eliminated})
		}
	}
	for _, state := range f.spaces {
		if state.visibility[player_id] > 0 {
			value := gin.H{"coordinate_key": state.key, "terrain_id": state.terrain_id}
			if state.visibility.see(player_id, state.owner) {
				value["ownership"] = state.owner
			}
			spaces = append(spaces, value)
		}
	}
	for _, effect := range f.effects {
		if value := effect.toFrontend(player_id); value != nil {
			effects = append(effects, value)
		}
	}
	return gin.H{
		"tick": f.tick, "terminal": f.terminal, "phase": f.phase,
		"units": units, "buildings": buildings, "resources": resources,
		"players": players, "spaces": spaces, "effects": effects,
	}
}

func (j *turnJournal) toFrontend(player_id int) gin.H {
	frames, boundaries := []gin.H{}, []gin.H{}
	units := map[uint64]tickUnitState{}
	buildings := map[uint64]tickBuildingState{}
	for _, frame := range append([]tickFrame{j.baseline}, j.frames...) {
		for _, state := range frame.units {
			if state.visibility.inspect(player_id, state.target.player_id) {
				units[state.target.internal_id] = state
			}
		}
		for _, state := range frame.buildings {
			if state.visibility.inspect(player_id, state.target.player_id) {
				buildings[state.target.internal_id] = state
			}
		}
	}
	catalog, building_catalog := []gin.H{}, []gin.H{}
	for id, state := range units {
		value := state.toFrontend()
		value["tick_actions"] = tickActionsToFrontend(j.histories[tickActorKey{"unit", id}], player_id)
		catalog = append(catalog, value)
	}
	for id, state := range buildings {
		value := state.toFrontend()
		value["tick_actions"] = tickActionsToFrontend(j.histories[tickActorKey{"building", id}], player_id)
		building_catalog = append(building_catalog, value)
	}
	for _, frame := range j.frames {
		frames = append(frames, frame.toFrontend(player_id))
	}
	for _, frame := range j.boundaries {
		boundaries = append(boundaries, frame.toFrontend(player_id))
	}
	return gin.H{
		"schema_version": 1, "turn_number": j.turn, "tick_count": j.tick_count,
		"baseline": j.baseline.toFrontend(player_id), "frames": frames,
		"units": catalog, "buildings": building_catalog, "boundary_events": boundaries,
	}
}
