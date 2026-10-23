package risq

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
)

type PathNode struct {
	zone            *RisqZone
	parent          *PathNode
	cost_from_start uint
	heuristic       uint
	total           uint
	depth           uint
}

// Zone moves auto-targeting will walk to reach a target before giving up on it
const nearbyPathDepth = 10

func (u *RisqUnit) findPath(target *RisqZone, attack_range defs.RisqRange) *MoveIntent {
	return u.findPathWithin(target, attack_range, 0, false)
}

func (u *RisqUnit) findAttackPath(target *RisqZone) *MoveIntent {
	move := u.findPathWithin(target, u.attack_range, 0, true)
	if move != nil {
		move.attacking = true
	}
	return move
}

// A max_depth of 0 searches without limit
func (u *RisqUnit) findPathWithin(target *RisqZone, attack_range defs.RisqRange, max_depth uint, attacking bool) *MoveIntent {
	start, is_garrisoned := u.pathStart()
	if start == nil || target == nil {
		return nil
	}
	goal, remaining := pathGoal(u.config, target, attack_range)
	if goal(start) {
		if is_garrisoned {
			return &MoveIntent{path: []*RisqZone{start}, next_step: start, intra_step: true, cost: start.space.moveCost().Intra_cost}
		}
		return nil
	}
	var facing func(*RisqZone) uint
	if max_depth == 0 {
		facing = func(z *RisqZone) uint { return facingDistance(z, target) }
	}
	blocked := func(from *RisqZone, to *RisqZone) bool {
		return !(attacking && to == target) && to.knownWallBlocksEntry(from, u.player_id)
	}
	return aStarPath(start, remaining, goal, max_depth, facing, blocked)
}

func facingDistance(z *RisqZone, target *RisqZone) uint {
	if z.adjacent_space != nil {
		return z.adjacent_space.distanceTo(target.space)
	}
	return z.space.distanceTo(target.space)
}

func (u *RisqUnit) canReach(target *RisqZone, attack_range defs.RisqRange) bool {
	start, _ := u.pathStart()
	return start == target || u.findPath(target, attack_range) != nil
}

func (u *RisqUnit) canReachToAttack(target *RisqZone) bool {
	start, _ := u.pathStart()
	return start == target || u.findAttackPath(target) != nil
}

func (u *RisqUnit) canReachNearby(target *RisqZone) bool {
	return u.inAttackRange(target) || u.findPathWithin(target, u.attack_range, nearbyPathDepth, true) != nil
}

func pathGoal(config *defs.Config, target *RisqZone, attack_range defs.RisqRange) (func(*RisqZone) bool, func(*RisqZone) uint) {
	space_range, ranged := attack_range.SpaceRadius()
	if !ranged {
		return func(z *RisqZone) bool { return z == target },
			func(z *RisqZone) uint {
				return z.space.distanceTo(target.space) * config.LowestInterMoveCost
			}
	}
	goal := func(z *RisqZone) bool {
		return rangeCovers(z, target, attack_range)
	}
	reach := max(space_range, 1)
	remaining := func(z *RisqZone) uint {
		dist := z.space.distanceTo(target.space)
		if dist <= reach {
			return 0
		}
		return (dist - reach) * config.LowestInterMoveCost
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

func aStarPath(start *RisqZone, remaining func(*RisqZone) uint, goal func(*RisqZone) bool, max_depth uint, facing func(*RisqZone) uint, blocked func(from *RisqZone, to *RisqZone) bool) *MoveIntent {
	var closest *PathNode
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
		if facing != nil && (closest == nil || closerNode(n, closest, facing)) {
			closest = n
		}
		if max_depth > 0 && n.depth >= max_depth {
			continue
		}
		for _, neighbor := range z.adjacent_zones {
			if neighbor.space.impassable() || blocked(z, neighbor) {
				continue
			}
			cost := neighbor.space.moveCost()
			move_cost := cost.Intra_cost
			if neighbor.space != z.space {
				move_cost = cost.Inter_cost
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
			neighbor_n.depth = n.depth + 1
			neighbor_n.cost_from_start = neighbor_g
			h_score := remaining(neighbor)
			neighbor_n.heuristic = h_score
			neighbor_n.total = neighbor_g + h_score
		}
	}

	if closest != nil {
		return closest.constructMoveIntent()
	}
	return nil
}

func closerNode(a *PathNode, b *PathNode, facing func(*RisqZone) uint) bool {
	if a.heuristic != b.heuristic {
		return a.heuristic < b.heuristic
	}
	if fa, fb := facing(a.zone), facing(b.zone); fa != fb {
		return fa < fb
	}
	return a.cost_from_start < b.cost_from_start
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
	cost := next_step.space.moveCost()
	move_cost := cost.Inter_cost
	if intra_step {
		move_cost = cost.Intra_cost
	}
	return &MoveIntent{
		path:       path,
		next_step:  next_step,
		intra_step: intra_step,
		cost:       move_cost,
	}
}
