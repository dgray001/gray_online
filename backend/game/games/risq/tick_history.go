package risq

import (
	"reflect"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

type tickActorKey struct {
	kind string
	id   uint64
}

type tickUnitState struct {
	target                tickTarget
	unit_id               uint32
	unit_type             defs.UnitType
	display_name          string
	location              tickLocation
	cs                    RisqCombatStats
	stamina, turn_stamina int
	deleted               bool
	visibility            tickVisibility
}

type tickBuildingState struct {
	target                                                       tickTarget
	building_id                                                  uint32
	display_name                                                 string
	location                                                     tickLocation
	cs                                                           RisqCombatStats
	stamina, turn_stamina, stamina_remaining, construction_total int
	under_construction                                           bool
	resources_left                                               float64
	renew_stamina_remaining                                      int
	deleted                                                      bool
	visibility                                                   tickVisibility
}

type tickResourceState struct {
	target      tickTarget
	resource_id uint32
	location    tickLocation
	remaining   float64
	deleted     bool
	visibility  tickVisibility
}

type tickPlayerState struct {
	player_id  int
	resources  defs.RisqResourceCost
	techs      map[uint32]bool
	eliminated bool
}

type tickSpaceState struct {
	key        uint
	terrain_id uint32
	owner      int
	visibility tickVisibility
}

type tickFrame struct {
	tick      uint16
	terminal  bool
	phase     string
	units     []tickUnitState
	buildings []tickBuildingState
	resources []tickResourceState
	players   []tickPlayerState
	spaces    []tickSpaceState
	effects   []tickEffect
}

type turnJournal struct {
	next_effect_id uint64

	turn, tick, tick_count uint16
	recording              bool
	baseline               tickFrame
	frames, boundaries     []tickFrame
	histories              map[tickActorKey][]*TickAction
	units                  map[uint64]tickUnitState
	buildings              map[uint64]tickBuildingState
	resources              map[uint64]tickResourceState
	players                map[int]tickPlayerState
	spaces                 map[uint]tickSpaceState
	effects                []tickEffect
}

func (r *GameRisq) beginTickHistory() {
	r.tick_history = &turnJournal{
		turn: r.turn_number, recording: true,
		histories: map[tickActorKey][]*TickAction{}, units: map[uint64]tickUnitState{},
		buildings: map[uint64]tickBuildingState{}, resources: map[uint64]tickResourceState{},
		players: map[int]tickPlayerState{}, spaces: map[uint]tickSpaceState{},
	}
	for _, unit := range r.units {
		unit.tick_actions, unit.tick_action, unit.garrison_tick_action = nil, nil, nil
	}
	for _, building := range r.buildings {
		building.tick_actions, building.tick_action, building.garrison_tick_action = nil, nil, nil
	}
}

func (r *GameRisq) beginTickHistoryPass() {
	r.tick_history.tick = r.current_tick + 1
	r.tick_history.effects = nil
	for _, unit := range r.units {
		unit.tick_action, unit.garrison_tick_action = nil, nil
	}
	for _, building := range r.buildings {
		building.tick_action = nil
	}
}

func snapshotTickUnit(unit *RisqUnit) tickUnitState {
	return tickUnitState{
		target: tickActorTarget(unit), unit_id: unit.unit_id, unit_type: unit.unitType(),
		display_name: unit.display_name, location: tickActorLocation(unit), cs: unit.cs,
		stamina: unit.current_stamina, turn_stamina: unit.turn_stamina,
		deleted: unit.deleted, visibility: tickZoneVisibility(tickActorZone(unit)),
	}
}

func snapshotTickBuilding(building *RisqBuilding) tickBuildingState {
	return tickBuildingState{
		target: tickActorTarget(building), building_id: building.building_id,
		display_name: building.display_name, location: tickActorLocation(building), cs: building.cs,
		stamina: building.current_stamina, turn_stamina: building.turn_stamina,
		stamina_remaining: building.stamina_remaining, construction_total: building.construction_stamina_total,
		under_construction: building.underConstruction(), resources_left: building.resources_left,
		renew_stamina_remaining: building.renew_stamina_remaining, deleted: building.deleted,
		visibility: tickZoneVisibility(building.zone),
	}
}

func (r *GameRisq) captureTickFrame(tick uint16, terminal bool, phase string) tickFrame {
	journal := r.tick_history
	frame := tickFrame{tick: tick, terminal: terminal, phase: phase, effects: journal.effects}
	for id, unit := range r.units {
		state := snapshotTickUnit(unit)
		if previous, ok := journal.units[id]; !ok || !reflect.DeepEqual(previous, state) {
			frame.units = append(frame.units, state)
		}
		journal.units[id] = state
	}
	for id, building := range r.buildings {
		state := snapshotTickBuilding(building)
		if previous, ok := journal.buildings[id]; !ok || !reflect.DeepEqual(previous, state) {
			frame.buildings = append(frame.buildings, state)
		}
		journal.buildings[id] = state
	}
	seen := map[uint64]bool{}
	for _, space := range r.allSpaces() {
		state := tickSpaceState{space.coordinate_key, space.terrain_id, space.ownership, tickZoneVisibility(space.getCenterZone())}
		if previous, ok := journal.spaces[state.key]; !ok || !reflect.DeepEqual(previous, state) {
			frame.spaces = append(frame.spaces, state)
		}
		journal.spaces[state.key] = state
		for id, resource := range space.resources {
			state := tickResourceState{
				target: tickTarget{"resource", id, -1}, resource_id: resource.resource_id,
				location: tickZoneLocation(resource.zone), remaining: resource.resources_left,
				visibility: tickZoneVisibility(resource.zone),
			}
			if previous, ok := journal.resources[id]; !ok || !reflect.DeepEqual(previous, state) {
				frame.resources = append(frame.resources, state)
			}
			journal.resources[id], seen[id] = state, true
		}
	}
	for id, previous := range journal.resources {
		if !seen[id] && !previous.deleted {
			previous.deleted, previous.remaining = true, 0
			journal.resources[id] = previous
			frame.resources = append(frame.resources, previous)
		}
	}
	for _, player := range r.players {
		resources := player.resources
		state := tickPlayerState{
			player_id: player.player.Player_id,
			resources: defs.RisqResourceCost{Food: resources.food, Wood: resources.wood, Stone: resources.stone, Gold: resources.gold},
			techs:     map[uint32]bool{}, eliminated: player.eliminated,
		}
		for id, researched := range player.researched_techs {
			state.techs[id] = researched
		}
		if previous, ok := journal.players[state.player_id]; !ok || !reflect.DeepEqual(previous, state) {
			frame.players = append(frame.players, state)
		}
		journal.players[state.player_id] = state
	}
	journal.effects = nil
	return frame
}

func (r *GameRisq) captureTickBaseline() {
	r.tick_history.baseline = r.captureTickFrame(0, false, "baseline")
}

func (r *GameRisq) captureTickEnd(terminal bool) {
	tick := r.current_tick
	if terminal {
		tick = r.tick_history.tick
	} else {
		r.tick_history.tick_count = tick
	}
	r.tick_history.frames = append(r.tick_history.frames, r.captureTickFrame(tick, terminal, "tick"))
}

func (r *GameRisq) captureTickBoundary(phase string) {
	if r.tick_history != nil {
		frame := r.captureTickFrame(r.tick_history.tick_count, false, phase)
		r.tick_history.boundaries = append(r.tick_history.boundaries, frame)
	}
}
