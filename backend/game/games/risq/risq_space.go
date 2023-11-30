package risq

import (
	"fmt"
	"os"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

type RisqSpace struct {
	coordinate game_utils.Coordinate2D
	zones      [][]*RisqZone
	visibility map[int]uint8
}

func createRisqSpace(i int, j int) *RisqSpace {
	space := RisqSpace{
		coordinate: game_utils.Coordinate2D{X: i, Y: j},
		visibility: make(map[int]uint8),
	}
	space.zones = make([][]*RisqZone, 3)
	for j := range space.zones {
		r := j - 1
		l := 3 - util.AbsInt(r)
		space.zones[j] = make([]*RisqZone, l)
		for i := range space.zones[j] {
			q := max(-1, -(1+r)) + i
			space.zones[j][i] = createRisqZone(q, r)
		}
	}
	return &space
}

func (s *RisqSpace) coordinateToIndex(c *game_utils.Coordinate2D) *game_utils.Coordinate2D {
	return &game_utils.Coordinate2D{
		X: c.Y + 1,
		Y: c.X - max(-1, -(1+c.Y)),
	}
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
	if zone.building != nil {
		fmt.Fprintln(os.Stderr, "Can't set building when building already there")
		return
	}
	zone.building = building
	s.visibility[building.player_id] = 1
}

func (s *RisqSpace) setUnit(c *game_utils.Coordinate2D, unit *RisqUnit) {
	zone := s.getZone(c)
	if zone == nil {
		fmt.Fprintln(os.Stderr, "Invalid zone coordinate: ", c.X, c.Y)
		return
	}
	zone.units[unit.internal_id] = unit
	s.visibility[unit.player_id] = 1
}

func (s *RisqSpace) getVisibility(player_id int) uint8 {
	v, ok := s.visibility[player_id]
	if ok {
		return v
	} else {
		return 0
	}
}

func (s *RisqSpace) toFrontend(player_id int, is_viewer bool) gin.H {
	space := gin.H{
		"coordinate": s.coordinate.ToFrontend(),
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
	return space
}
