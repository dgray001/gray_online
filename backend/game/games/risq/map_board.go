package risq

import (
	"math/rand"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/mapgen"
)

type mapBoard struct {
	r *GameRisq
}

type mapSpace struct {
	r *GameRisq
	s *RisqSpace
}

type mapZone struct {
	r *GameRisq
	z *RisqZone
}

func (b mapBoard) wrapSpace(s *RisqSpace) mapgen.Space {
	if s == nil {
		return nil
	}
	return mapSpace{b.r, s}
}

func (b mapBoard) wrapZone(z *RisqZone) mapgen.Zone {
	if z == nil {
		return nil
	}
	return mapZone{b.r, z}
}

func (b mapBoard) Allocate(board_size uint16) {
	b.r.allocateBoard(board_size)
}

func (b mapBoard) BoardSize() uint16 {
	return b.r.board_size
}

func (b mapBoard) Spaces() []mapgen.Space {
	spaces := make([]mapgen.Space, 0)
	for _, row := range b.r.spaces {
		for _, space := range row {
			if space != nil {
				spaces = append(spaces, mapSpace{b.r, space})
			}
		}
	}
	return spaces
}

func (b mapBoard) Space(c game_utils.Coordinate2D) mapgen.Space {
	return b.wrapSpace(b.r.getSpace(&c))
}

func (b mapBoard) RemoveSpace(s mapgen.Space) {
	b.r.removeSpace(s.(mapSpace).s)
}

func (b mapBoard) AddRegion(name string, coordinate_keys map[uint]bool) error {
	return b.r.addRegion(name, 0, coordinate_keys)
}

func (b mapBoard) PlaceResource(z mapgen.Zone, resource_id uint32) {
	zone := z.(mapZone).z
	zone.space.setResource(&zone.coordinate, createRisqResource(b.r.nextResourceInternalId(), resource_id))
}

func (b mapBoard) PlaceBuilding(z mapgen.Zone, building_id uint32, player_index int) bool {
	zone := z.(mapZone).z
	player := b.r.players[player_index]
	building := createRisqBuilding(b.r.nextBuildingInternalId(), building_id, player.player.Player_id)
	zone.space.setBuilding(&zone.coordinate, building)
	if zone.building != building {
		return false
	}
	player.buildings[building.internal_id] = building
	b.r.buildings[building.internal_id] = building
	return true
}

func (b mapBoard) PlaceUnit(z mapgen.Zone, unit_id uint32, player_index int) {
	zone := z.(mapZone).z
	player := b.r.players[player_index]
	unit := createRisqUnit(b.r.nextUnitInternalId(), unit_id, player)
	zone.space.setUnit(&zone.coordinate, unit)
	player.units[unit.internal_id] = unit
	b.r.units[unit.internal_id] = unit
}

func (b mapBoard) SetStartingBank(player_index int, bank mapgen.StartingBank) {
	resources := b.r.players[player_index].resources
	resources.food, resources.wood, resources.stone, resources.gold = bank.Food, bank.Wood, bank.Stone, bank.Gold
}

func (s mapSpace) Key() uint {
	return s.s.coordinate_key
}

func (s mapSpace) Coordinate() game_utils.Coordinate2D {
	return s.s.coordinate
}

func (s mapSpace) Terrain() uint32 {
	return s.s.terrain_id
}

func (s mapSpace) SetTerrain(terrain_id uint32) {
	s.s.terrain_id = terrain_id
}

func (s mapSpace) Impassable() bool {
	return s.s.impassable()
}

func (s mapSpace) SortedAdjacent() []mapgen.Space {
	adjacent := s.s.sortedAdjacentSpaces()
	spaces := make([]mapgen.Space, len(adjacent))
	for i, a := range adjacent {
		spaces[i] = mapSpace{s.r, a}
	}
	return spaces
}

func (s mapSpace) Zones() []mapgen.Zone {
	zones := make([]mapgen.Zone, 0)
	for _, row := range s.s.zones {
		for _, zone := range row {
			zones = append(zones, mapZone{s.r, zone})
		}
	}
	return zones
}

func (s mapSpace) Zone(local game_utils.Coordinate2D) mapgen.Zone {
	return mapBoard{s.r}.wrapZone(s.s.getZone(&local))
}

func (s mapSpace) CenterZone() mapgen.Zone {
	return mapZone{s.r, s.s.getCenterZone()}
}

func (s mapSpace) ShuffledEdgeZones(rng *rand.Rand) []mapgen.Zone {
	shuffled := s.s.getZonesAsRandomArray(false, rng)
	zones := make([]mapgen.Zone, len(shuffled))
	for i, zone := range shuffled {
		zones[i] = mapZone{s.r, zone}
	}
	return zones
}

func (s mapSpace) ClearOccupants() {
	for _, row := range s.s.zones {
		for _, zone := range row {
			zone.terrain_override = 0
			if zone.resource != nil {
				s.s.removeResource(zone.resource)
			}
			if zone.building != nil {
				building := zone.building
				s.s.removeBuilding(building)
				delete(s.r.buildings, building.internal_id)
				delete(s.r.players[building.player_id].buildings, building.internal_id)
			}
		}
	}
}

func (z mapZone) Key() uint {
	return z.z.coordinate_key
}

func (z mapZone) Local() game_utils.Coordinate2D {
	return z.z.coordinate
}

func (z mapZone) Space() mapgen.Space {
	return mapSpace{z.r, z.z.space}
}

func (z mapZone) IsCenter() bool {
	return z.z.isCenter()
}

func (z mapZone) Adjacent() []mapgen.Zone {
	zones := make([]mapgen.Zone, len(z.z.adjacent_zones))
	for i, adj := range z.z.adjacent_zones {
		zones[i] = mapZone{z.r, adj}
	}
	return zones
}

func (z mapZone) Occupied() bool {
	return z.z.resource != nil || z.z.building != nil
}

func (z mapZone) ResourceId() (uint32, bool) {
	if z.z.resource == nil {
		return 0, false
	}
	return z.z.resource.resource_id, true
}

func (z mapZone) RemoveResource() {
	delete(z.z.space.resources, z.z.resource.internal_id)
	z.z.resource = nil
}

func (z mapZone) SetTerrainOverride(terrain_id uint32) {
	z.z.terrain_override = terrain_id
}
