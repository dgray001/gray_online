package risq

import (
	"github.com/dgray001/gray_online/game/game_utils"
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
	r.spaces = make([][]*RisqSpace, 2*int(board_size)+1)
	for j := range r.spaces {
		row_r := j - int(board_size)
		l := 2*int(board_size) + 1 - util.AbsInt(row_r)
		r.spaces[j] = make([]*RisqSpace, l)
		for i := range r.spaces[j] {
			q := max(-int(board_size), -(int(board_size)+row_r)) + i
			r.spaces[j][i] = createRisqSpace(q, row_r, defaultTerrainId)
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

func (r *GameRisq) removeSpace(space *RisqSpace) {
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
