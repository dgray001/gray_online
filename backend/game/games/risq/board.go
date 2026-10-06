package risq

import (
	"fmt"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
)

func (r *GameRisq) coordinateToIndex(c *game_utils.Coordinate2D) *game_utils.Coordinate2D {
	return &game_utils.Coordinate2D{
		X: c.Y + int(r.board_size),
		Y: c.X - max(-int(r.board_size), -(int(r.board_size)+c.Y)),
	}
}

func (r *GameRisq) getSpace(c *game_utils.Coordinate2D) *RisqSpace {
	index := r.coordinateToIndex(c)
	if index.X < 0 || index.X >= len(r.spaces) {
		return nil
	}
	row := r.spaces[index.X]
	if index.Y < 0 || index.Y >= len(row) {
		return nil
	}
	return row[index.Y]
}

func (r *GameRisq) allSpaces() []*RisqSpace {
	spaces := make([]*RisqSpace, 0)
	for _, row := range r.spaces {
		for _, space := range row {
			if space != nil {
				spaces = append(spaces, space)
			}
		}
	}
	return spaces
}

func (r *GameRisq) allocateBoard(board_size uint16) {
	r.board_size = board_size
	r.space_distances = nil
	r.spaces = make([][]*RisqSpace, 2*int(board_size)+1)
	for j := range r.spaces {
		row_r := j - int(board_size)
		l := 2*int(board_size) + 1 - util.AbsInt(row_r)
		r.spaces[j] = make([]*RisqSpace, l)
		for i := range r.spaces[j] {
			q := max(-int(board_size), -(int(board_size)+row_r)) + i
			r.spaces[j][i] = createRisqSpace(q, row_r, defs.DefaultTerrainId)
		}
	}
	for _, row := range r.spaces {
		for _, space := range row {
			for _, v := range game_utils.AxialDirectionVectors() {
				adjacent := r.getSpace(space.coordinate.Add(&v))
				if adjacent != nil {
					space.setAdjacentSpace(adjacent, &v)
				}
			}
		}
	}
}

func removeZoneFromSlice(zones []*RisqZone, target *RisqZone) []*RisqZone {
	for i, z := range zones {
		if z == target {
			return append(zones[:i], zones[i+1:]...)
		}
	}
	return zones
}

// Joins first's edge in direction (an index into AxialDirectionVectors) to second's opposite edge like a normal border
func connectSeam(first *RisqSpace, second *RisqSpace, direction int) error {
	vectors := game_utils.AxialDirectionVectors()
	if direction < 0 || direction >= len(vectors) {
		return fmt.Errorf("seam direction %d is not 0 to 5", direction)
	}
	vector := vectors[direction]
	inverted := vector.Invert()
	if first.getZone(&vector).adjacent_space != nil || second.getZone(inverted).adjacent_space != nil {
		return fmt.Errorf("seam edge between %s and %s is already connected", first.coordinate.ToString(), second.coordinate.ToString())
	}
	first.setAdjacentSpace(second, &vector)
	second.setAdjacentSpace(first, inverted)
	return nil
}

func (r *GameRisq) removeSpace(space *RisqSpace) {
	r.space_distances = nil
	for _, v := range game_utils.AxialDirectionVectors() {
		zone := space.getZone(&v)
		if zone == nil || zone.adjacent_space == nil {
			continue
		}
		neighbor := zone.adjacent_space
		inverted := v.Invert()
		if neighbor_zone := neighbor.getZone(inverted); neighbor_zone != nil {
			neighbor_zone.adjacent_zones = removeZoneFromSlice(neighbor_zone.adjacent_zones, zone)
			neighbor_zone.adjacent_space = nil
		}
		delete(neighbor.adjacent_spaces, util.Pair(inverted.X, inverted.Y))
	}
	space.adjacent_spaces = make(map[uint]*RisqSpace)
	index := r.coordinateToIndex(&space.coordinate)
	if index.X < 0 || index.X >= len(r.spaces) {
		return
	}
	row := r.spaces[index.X]
	if index.Y < 0 || index.Y >= len(row) {
		return
	}
	row[index.Y] = nil
}
