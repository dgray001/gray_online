package aibridge

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/ai"
)

const unreachableSpaceDistance = 1 << 15

// Normal borders between coordinate neighbors present on the board, plus the snapshot's seams and links
func (v *aiView) spaceNeighbors(space ai.Coordinate) []ai.Coordinate {
	neighbors := make([]ai.Coordinate, 0, 6)
	for _, direction := range game_utils.AxialDirectionVectors() {
		adjacent := ai.Coordinate{X: space.X + direction.X, Y: space.Y + direction.Y}
		if v.spaces[adjacent] != nil {
			neighbors = append(neighbors, adjacent)
		}
	}
	return append(neighbors, v.space_links[space]...)
}

func (v *aiView) indexSpaceLinks() {
	v.space_links = map[ai.Coordinate][]ai.Coordinate{}
	for _, link := range v.game.SpaceLinks {
		from, to := toCoordinate(link.From), toCoordinate(link.To)
		v.space_links[from] = append(v.space_links[from], to)
		v.space_links[to] = append(v.space_links[to], from)
	}
}

// Space-to-space steps over the board graph; memoized per source space for the life of the view
func (v *aiView) SpaceDistance(from ai.Coordinate, to ai.Coordinate) int {
	row, ok := v.space_distances[from]
	if !ok {
		row = map[ai.Coordinate]int{from: 0}
		queue := []ai.Coordinate{from}
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			for _, adjacent := range v.spaceNeighbors(current) {
				if _, seen := row[adjacent]; !seen {
					row[adjacent] = row[current] + 1
					queue = append(queue, adjacent)
				}
			}
		}
		v.space_distances[from] = row
	}
	if distance, reachable := row[to]; reachable {
		return distance
	}
	return unreachableSpaceDistance
}

// Zones within a space are one step apart unless opposite; crossing into another space costs more
func (v *aiView) LocationDistance(a ai.ZoneRef, b ai.ZoneRef) int {
	if a.Space == b.Space {
		return axialDistance(a.Zone, b.Zone)
	}
	return v.SpaceDistance(a.Space, b.Space) * 6
}
