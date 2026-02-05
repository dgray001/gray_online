package risq

import (
	"fmt"
	"os"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

type TerrainType uint8

const (
	TerrainType_FLATLANDS TerrainType = iota
	TerrainType_HILLY
	TerrainType_MOUNTAINOUS
	TerrainType_VALLEY
	TerrainType_SWAMP
	TerrainType_SHALLOWS
	TerrainType_WATER
	TerrainType_DEEP_WATER
)

type RisqSpace struct {
	terrain         TerrainType
	coordinate      game_utils.Coordinate2D
	coordinate_key  uint
	zones           [][]*RisqZone
	resources       map[uint64]*RisqResource
	buildings       map[uint64]*RisqBuilding
	units           map[uint64]*RisqUnit
	visibility      map[int]uint8
	adjacent_spaces map[uint]*RisqSpace
}

func createRisqSpace(q int, r int, terrain TerrainType) *RisqSpace {
	space := RisqSpace{
		terrain:         terrain,
		coordinate:      game_utils.Coordinate2D{X: q, Y: r},
		coordinate_key:  util.Pair(q, r),
		resources:       make(map[uint64]*RisqResource),
		buildings:       make(map[uint64]*RisqBuilding),
		units:           make(map[uint64]*RisqUnit),
		visibility:      make(map[int]uint8),
		adjacent_spaces: make(map[uint]*RisqSpace),
	}
	space.zones = make([][]*RisqZone, 3)
	for j := range space.zones {
		r_zone := j - 1
		l := 3 - util.AbsInt(r_zone)
		space.zones[j] = make([]*RisqZone, l)
		for i := range space.zones[j] {
			q_zone := max(-1, -(1+r_zone)) + i
			space.zones[j][i] = createRisqZone(q_zone, r_zone, &space)
		}
	}
	return &space
}

func (s *RisqSpace) setAdjacentSpace(adj *RisqSpace, v *game_utils.Coordinate2D) {
	zone := s.getZone(v)
	if zone == nil {
		fmt.Fprintln(os.Stderr, "Invalid zone coordinate: ", v.X, v.Y)
		return
	}
	if zone.adjacent_space != nil {
		fmt.Fprintln(os.Stderr, "Already set adjacent space for this zone: ", v.X, v.Y)
		return
	}
	zone.adjacent_space = adj
	s.adjacent_spaces[util.Pair(v.X, v.Y)] = adj
}

func (s *RisqSpace) coordinateToIndex(c *game_utils.Coordinate2D) *game_utils.Coordinate2D {
	return &game_utils.Coordinate2D{
		X: c.Y + 1,
		Y: c.X - max(-1, -(1+c.Y)),
	}
}

func (s *RisqSpace) getZonesAsRandomArray(include_middle bool) []*RisqZone {
	zones := make([]*RisqZone, 0)
	for i, row := range s.zones {
		for j, zone := range row {
			if include_middle || i != 1 || j != 1 {
				zones = append(zones, zone)
			}
		}
	}
	return util.Shuffle(zones)
}

func (s *RisqSpace) getZone(c *game_utils.Coordinate2D) *RisqZone {
	index := s.coordinateToIndex(c)
	if index.X < 0 || index.X >= len(s.zones) {
		return nil
	}
	row := s.zones[index.X]
	if index.Y < 0 || index.Y >= len(row) {
		return nil
	}
	return row[index.Y]
}

func (s *RisqSpace) setBuilding(c *game_utils.Coordinate2D, building *RisqBuilding) {
	zone := s.getZone(c)
	if zone == nil {
		fmt.Fprintln(os.Stderr, "Invalid zone coordinate: ", c.X, c.Y)
		return
	}
	if zone.resource != nil && zone.resource.resources_left > 0 {
		fmt.Fprintln(os.Stderr, "Can't set building when resource already there")
		return
	}
	if zone.building != nil && !zone.building.deleted {
		fmt.Fprintln(os.Stderr, "Can't set building when building already there")
		return
	}
	s.buildings[building.internal_id] = building
	zone.building = building
	building.zone = zone
	s.addVision(building.vision(), zone, building.player_id)
}

func (s *RisqSpace) setUnit(c *game_utils.Coordinate2D, unit *RisqUnit) {
	zone := s.getZone(c)
	if zone == nil {
		fmt.Fprintln(os.Stderr, "Invalid zone coordinate: ", c.X, c.Y)
		return
	}
	s.units[unit.internal_id] = unit
	zone.units[unit.internal_id] = unit
	unit.zone = zone
	s.addVision(unit.vision(), zone, unit.player_id)
}

func (s *RisqSpace) addVision(v *RisqVision, z *RisqZone, player_id int) {
	checked := make(map[uint]bool)
	if s.getVisibility(player_id) < v.space {
		s.visibility[player_id] = v.space
	}
	checked[s.coordinate_key] = true
	if z.isCenter() {
		for _, adj := range s.adjacent_spaces {
			if adj.getVisibility(player_id) < v.adjacent {
				adj.visibility[player_id] = v.adjacent
			}
			checked[adj.coordinate_key] = true
		}
	} else {
		// TODO: implement with edge_adjacent and edge_opposite
		for _, adj := range s.adjacent_spaces {
			if adj.getVisibility(player_id) < v.adjacent {
				adj.visibility[player_id] = v.adjacent
			}
			checked[adj.coordinate_key] = true
		}
	}
	for _, adj := range s.adjacent_spaces {
		for _, sec := range adj.adjacent_spaces {
			if util.MapContains(checked, sec.coordinate_key) {
				continue
			}
			if sec.getVisibility(player_id) < v.secondary {
				sec.visibility[player_id] = v.secondary
			}
			checked[sec.coordinate_key] = true
		}
	}
}

func (s *RisqSpace) setResource(c *game_utils.Coordinate2D, resource *RisqResource) {
	zone := s.getZone(c)
	if zone == nil {
		fmt.Fprintln(os.Stderr, "Invalid zone coordinate: ", c.X, c.Y)
		return
	}
	if zone.resource != nil && zone.resource.resources_left > 0 {
		fmt.Fprintln(os.Stderr, "Can't set resource when resource already there")
		return
	}
	if zone.building != nil && !zone.building.deleted {
		fmt.Fprintln(os.Stderr, "Can't set resource when building already there")
		return
	}
	s.resources[resource.internal_id] = resource
	zone.resource = resource
	resource.zone = zone
}

func (s *RisqSpace) getVisibility(player_id int) uint8 {
	v, ok := s.visibility[player_id]
	if ok {
		return v
	} else {
		return 0
	}
}

func (s *RisqSpace) toFrontend(player_id int, _ bool) gin.H {
	space := gin.H{
		"terrain":        s.terrain,
		"coordinate":     s.coordinate.ToFrontend(),
		"coordinate_key": s.coordinate_key,
	}
	v := s.getVisibility(player_id)
	space["visibility"] = v
	if v < 1 {
		return space // unexplored
	}
	// TODO: visibility effects what you can see
	zones := [][]gin.H{}
	for _, row := range s.zones {
		zones_row := []gin.H{}
		for _, zone := range row {
			zones_row = append(zones_row, zone.toFrontend())
		}
		zones = append(zones, zones_row)
	}
	space["zones"] = zones
	resources := make([]gin.H, 0)
	for _, resource := range s.resources {
		if resource != nil && resource.resources_left > 0 {
			resources = append(resources, resource.toFrontend())
		}
	}
	space["resources"] = resources
	buildings := make([]gin.H, 0)
	for _, building := range s.buildings {
		if building != nil && !building.deleted {
			buildings = append(buildings, building.toFrontend())
		}
	}
	space["buildings"] = buildings
	units := make([]gin.H, 0)
	for _, unit := range s.units {
		if unit != nil && !unit.deleted {
			units = append(units, unit.toFrontend())
		}
	}
	space["units"] = units
	return space
}
