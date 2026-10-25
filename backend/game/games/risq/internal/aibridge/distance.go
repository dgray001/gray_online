package aibridge

import (
	"cmp"
	"slices"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/ai"
)

const unreachableSpaceDistance = 1 << 15

// Normal borders between coordinate neighbors present on the board, plus the snapshot's seams and links
func (v *aiView) SpaceNeighbors(space ai.Coordinate) []ai.Coordinate {
	neighbors := make([]ai.Coordinate, 0, 6)
	for _, direction := range game_utils.AxialDirectionVectors() {
		adjacent := ai.Coordinate{X: space.X + direction.X, Y: space.Y + direction.Y}
		if v.spaces[adjacent] != nil {
			neighbors = append(neighbors, adjacent)
		}
	}
	neighbors = append(neighbors, v.space_links[space]...)
	slices.SortFunc(neighbors, func(a, b ai.Coordinate) int {
		if a.X != b.X {
			return cmp.Compare(a.X, b.X)
		}
		return cmp.Compare(a.Y, b.Y)
	})
	return slices.Compact(neighbors)
}

func edgeZone(direction int) ai.Coordinate {
	vector := game_utils.AxialDirectionVectors()[direction]
	return ai.Coordinate{X: vector.X, Y: vector.Y}
}

// The zone across the border from an edge zone, when the board has a space there
func (v *aiView) facingZone(from ai.ZoneRef) (ai.ZoneRef, bool) {
	if linked, ok := v.linked_edges[from]; ok {
		return linked, true
	}
	neighbor := ai.Coordinate{X: from.Space.X + from.Zone.X, Y: from.Space.Y + from.Zone.Y}
	if game_utils.AxialDirectionIndex(game_utils.Coordinate2D{X: from.Zone.X, Y: from.Zone.Y}) < 0 || v.spaces[neighbor] == nil {
		return ai.ZoneRef{}, false
	}
	return ai.ZoneRef{Space: neighbor, Zone: ai.Coordinate{X: -from.Zone.X, Y: -from.Zone.Y}}, true
}

func (v *aiView) indexSpaceLinks() {
	v.space_links = map[ai.Coordinate][]ai.Coordinate{}
	v.linked_edges = map[ai.ZoneRef]ai.ZoneRef{}
	for _, link := range v.game.SpaceLinks {
		from, to := toCoordinate(link.From), toCoordinate(link.To)
		from_edge, to_edge := ai.ZoneRef{Space: from, Zone: edgeZone(link.Direction)}, ai.ZoneRef{Space: to, Zone: edgeZone(link.ToDirection)}
		v.linked_edges[from_edge], v.linked_edges[to_edge] = to_edge, from_edge
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
			for _, adjacent := range v.SpaceNeighbors(current) {
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
