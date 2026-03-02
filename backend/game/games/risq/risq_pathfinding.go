package risq

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
)

type PathNode struct {
	zone   *RisqZone
	parent *PathNode
	// Cost from start
	g uint
	// Estimated cost to target (heuristic)
	h uint
	// Total cost (g + h)
	f uint
}

func (u *RisqUnit) findPath(target *RisqZone) *MoveIntent {
	start := u.zone
	if start == target {
		return nil
	}
	open_set := []*RisqZone{start}
	tracker := make(map[*RisqZone]*PathNode)
	h_score := game_utils.AxialDistance(start.space.coordinate, target.space.coordinate) * 6
	tracker[start] = &PathNode{
		zone:   start,
		parent: nil,
		g:      0,
		h:      h_score,
		f:      h_score,
	}

	for len(open_set) > 0 {
		best_index := 0
		for i := 1; i < len(open_set); i++ {
			if tracker[open_set[i]].f < tracker[open_set[best_index]].f {
				best_index = i
			}
		}
		z := open_set[best_index]
		n := tracker[z]
		if z == target {
			return n.constructMoveIntent()
		}
		open_set = util.FastDelete(open_set, best_index)
		for _, neighbor := range z.adjacent_zones {
			move_cost := uint(1)
			if neighbor.space != z.space {
				move_cost = 6
			}
			neighbor_g := n.g + move_cost
			neighbor_n, visited := tracker[neighbor]
			if visited && neighbor_g >= neighbor_n.g {
				continue // shorter path to this node already found
			}
			if !visited {
				neighbor_n = &PathNode{zone: neighbor}
				tracker[neighbor] = neighbor_n
				open_set = append(open_set, neighbor)
			}
			neighbor_n.parent = n
			neighbor_n.g = neighbor_g
			h_score := game_utils.AxialDistance(neighbor.space.coordinate, target.space.coordinate) * 6
			neighbor_n.h = h_score
			neighbor_n.f = neighbor_g + h_score
		}
	}

	// Never reached target zone
	// TODO: move in direction of target if able
	return nil
}

func (n *PathNode) constructMoveIntent() *MoveIntent {
	path := make([]*RisqZone, 0)
	curr := n
	for curr != nil {
		path = append([]*RisqZone{curr.zone}, path...)
		curr = curr.parent
	}
	if len(path) < 2 {
		return nil // If no move is needed
	}
	return &MoveIntent{
		path:       path,
		next_step:  path[1],
		intra_step: path[0].space == path[1].space,
	}
}
