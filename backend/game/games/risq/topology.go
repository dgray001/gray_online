package risq

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
)

// Hop counts between every pair of spaces over adjacent_spaces, so seams and links count as one step.
type spaceDistances struct {
	hops [][]uint16
}

const unreachableSpaceDistance = 1 << 15

// Valid only until the space graph next changes; allocateBoard, removeSpace and ConnectSpaces clear it.
func (r *GameRisq) ensureSpaceDistances() {
	if r.space_distances != nil {
		return
	}
	spaces := r.allSpaces()
	distances := &spaceDistances{hops: make([][]uint16, len(spaces))}
	for i, space := range spaces {
		space.distances, space.distance_index = distances, i
		for _, row := range space.zones {
			for _, zone := range row {
				zone.range_zones = [6]map[*RisqZone]bool{}
			}
		}
	}
	neighbors := make([][]int, len(spaces))
	for i, space := range spaces {
		for _, adjacent := range space.adjacent_spaces {
			neighbors[i] = append(neighbors[i], adjacent.distance_index)
		}
	}
	queue := make([]int, 0, len(spaces))
	for i := range spaces {
		row := make([]uint16, len(spaces))
		for j := range row {
			row[j] = unreachableSpaceDistance
		}
		row[i] = 0
		queue = append(queue[:0], i)
		for head := 0; head < len(queue); head++ {
			current := queue[head]
			for _, adjacent := range neighbors[current] {
				if row[adjacent] == unreachableSpaceDistance {
					row[adjacent] = row[current] + 1
					queue = append(queue, adjacent)
				}
			}
		}
		distances.hops[i] = row
	}
	r.space_distances = distances
	r.space_links = spaceLinks(spaces)
}

// Every adjacency that isn't the normal border between coordinate neighbors, once per pair: links carry the
// direction (index into AxialDirectionVectors) leaving "from" and the one arriving at "to".
func spaceLinks(spaces []*RisqSpace) []spaceLink {
	links := []spaceLink{}
	for _, space := range spaces {
		for direction, vector := range game_utils.AxialDirectionVectors() {
			zone := space.getZone(&vector)
			adjacent := zone.adjacent_space
			if adjacent == nil || space.coordinate_key >= adjacent.coordinate_key {
				continue
			}
			to_direction := arrivalDirection(zone, adjacent)
			if adjacent.coordinate != *space.coordinate.Add(&vector) || to_direction != (direction+3)%6 {
				links = append(links, spaceLink{from: space, to: adjacent, direction: direction, to_direction: to_direction})
			}
		}
	}
	return links
}

// The edge of adjacent that zone's border joins
func arrivalDirection(zone *RisqZone, adjacent *RisqSpace) int {
	for _, other := range zone.adjacent_zones {
		if other.space == adjacent && !other.isCenter() {
			return other.direction
		}
	}
	return -1
}

// A link joining from's edge in direction to the edge of to in to_direction
type spaceLink struct {
	from, to     *RisqSpace
	direction    int
	to_direction int
}

func (l spaceLink) toFrontend() gin.H {
	return gin.H{"from": l.from.coordinate.ToFrontend(), "to": l.to.coordinate.ToFrontend(), "direction": l.direction, "to_direction": l.to_direction}
}

// Only the links a player has explored: at least one end is a space they know
func (r *GameRisq) spaceLinksToFrontend(player_id int) []gin.H {
	links := []gin.H{}
	for _, link := range r.space_links {
		if link.from.getVisibility(player_id) >= defs.VisibilityFog || link.to.getVisibility(player_id) >= defs.VisibilityFog {
			links = append(links, link.toFrontend())
		}
	}
	return links
}

// Number of space-to-space steps between two spaces; callers must have run ensureSpaceDistances
func (s *RisqSpace) distanceTo(other *RisqSpace) uint {
	if s == other {
		return 0
	}
	return uint(s.distances.hops[s.distance_index][other.distance_index])
}
