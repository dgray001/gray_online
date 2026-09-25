package risq

import (
	"fmt"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
)

type RisqRange uint8

const (
	RisqRange_NONE RisqRange = iota
	RisqRange_ZONE
	RisqRange_SPACE
	RisqRange_ADJACENT
	RisqRange_SECONDARY
	RisqRange_END
)

func parseRange(s string) (RisqRange, error) {
	switch s {
	case "", "zone":
		return RisqRange_ZONE, nil
	case "space":
		return RisqRange_SPACE, nil
	case "adjacent":
		return RisqRange_ADJACENT, nil
	case "secondary":
		return RisqRange_SECONDARY, nil
	default:
		return RisqRange_NONE, fmt.Errorf("unknown range %q", s)
	}
}

func (r RisqRange) spaceRadius() (uint, bool) {
	switch r {
	case RisqRange_SPACE:
		return 0, true
	case RisqRange_ADJACENT:
		return 1, true
	case RisqRange_SECONDARY:
		return 2, true
	default:
		return 0, false
	}
}

type PathNode struct {
	zone            *RisqZone
	parent          *PathNode
	cost_from_start uint
	heuristic       uint
	total           uint
}

func (u *RisqUnit) findPath(target *RisqZone, attack_range RisqRange) *MoveIntent {
	start, is_garrisoned := u.pathStart()
	if start == nil || target == nil {
		return nil
	}
	goal, remaining := pathGoal(target, attack_range)
	if goal(start) {
		if is_garrisoned {
			return &MoveIntent{path: []*RisqZone{start}, next_step: start, intra_step: true, cost: start.space.terrainType().moveCost().intra_cost}
		}
		return nil
	}
	return aStarPath(start, remaining, goal)
}

func (u *RisqUnit) canReach(target *RisqZone, attack_range RisqRange) bool {
	start, _ := u.pathStart()
	return start == target || u.findPath(target, attack_range) != nil
}

func pathGoal(target *RisqZone, attack_range RisqRange) (func(*RisqZone) bool, func(*RisqZone) uint) {
	space_range, ranged := attack_range.spaceRadius()
	if !ranged {
		return func(z *RisqZone) bool { return z == target },
			func(z *RisqZone) uint {
				return game_utils.AxialDistance(z.space.coordinate, target.space.coordinate) * lowestInterMoveCost
			}
	}
	goal := func(z *RisqZone) bool {
		return game_utils.AxialDistance(z.space.coordinate, target.space.coordinate) <= space_range
	}
	remaining := func(z *RisqZone) uint {
		dist := game_utils.AxialDistance(z.space.coordinate, target.space.coordinate)
		if dist <= space_range {
			return 0
		}
		return (dist - space_range) * lowestInterMoveCost
	}
	return goal, remaining
}

func (u *RisqUnit) pathStart() (*RisqZone, bool) {
	if u.zone != nil {
		return u.zone, false
	}
	if u.garrisoned_in != nil {
		return u.garrisoned_in.zone, true
	}
	return nil, false
}

func aStarPath(start *RisqZone, remaining func(*RisqZone) uint, goal func(*RisqZone) bool) *MoveIntent {
	open_set := []*RisqZone{start}
	tracker := make(map[*RisqZone]*PathNode)
	h_score := remaining(start)
	tracker[start] = &PathNode{
		zone:            start,
		cost_from_start: 0,
		heuristic:       h_score,
		total:           h_score,
	}

	for len(open_set) > 0 {
		best_index := 0
		for i := 1; i < len(open_set); i++ {
			if tracker[open_set[i]].total < tracker[open_set[best_index]].total {
				best_index = i
			}
		}
		z := open_set[best_index]
		n := tracker[z]
		if goal(z) {
			return n.constructMoveIntent()
		}
		open_set = util.FastDelete(open_set, best_index)
		for _, neighbor := range z.adjacent_zones {
			if neighbor.space.impassable() {
				continue
			}
			cost := neighbor.space.terrainType().moveCost()
			move_cost := cost.intra_cost
			if neighbor.space != z.space {
				move_cost = cost.inter_cost
			}
			neighbor_g := n.cost_from_start + move_cost
			neighbor_n, visited := tracker[neighbor]
			if visited && neighbor_g >= neighbor_n.cost_from_start {
				continue
			}
			if !visited {
				neighbor_n = &PathNode{zone: neighbor}
				tracker[neighbor] = neighbor_n
				open_set = append(open_set, neighbor)
			}
			neighbor_n.parent = n
			neighbor_n.cost_from_start = neighbor_g
			h_score := remaining(neighbor)
			neighbor_n.heuristic = h_score
			neighbor_n.total = neighbor_g + h_score
		}
	}

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
		return nil
	}
	next_step := path[1]
	intra_step := path[0].space == next_step.space
	cost := next_step.space.terrainType().moveCost()
	move_cost := cost.inter_cost
	if intra_step {
		move_cost = cost.intra_cost
	}
	return &MoveIntent{
		path:       path,
		next_step:  next_step,
		intra_step: intra_step,
		cost:       move_cost,
	}
}
