package risq

import (
	"github.com/dgray001/gray_online/game/game_utils"
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

// Every adjacency that isn't the normal border between coordinate neighbors, once per pair: seams carry the
// direction (index into AxialDirectionVectors) leaving "from".
func spaceLinks(spaces []*RisqSpace) []gin.H {
	links := []gin.H{}
	link := func(from *RisqSpace, to *RisqSpace, direction int) {
		links = append(links, gin.H{"from": from.coordinate.ToFrontend(), "to": to.coordinate.ToFrontend(), "direction": direction})
	}
	for _, space := range spaces {
		for direction, vector := range game_utils.AxialDirectionVectors() {
			adjacent := space.getZone(&vector).adjacent_space
			if adjacent != nil && adjacent.coordinate != *space.coordinate.Add(&vector) && space.coordinate_key < adjacent.coordinate_key {
				link(space, adjacent, direction)
			}
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
