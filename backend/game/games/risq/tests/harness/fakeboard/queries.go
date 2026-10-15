package fakeboard

import (
	"fmt"
	"slices"

	"github.com/dgray001/gray_online/game/game_utils"
)

// For each player, the spaces holding any of that player's buildings or units
func (b *Board) PlayerSpaces() map[int]map[game_utils.Coordinate2D]bool {
	spaces := map[int]map[game_utils.Coordinate2D]bool{}
	mark := func(player int, c game_utils.Coordinate2D) {
		if spaces[player] == nil {
			spaces[player] = map[game_utils.Coordinate2D]bool{}
		}
		spaces[player][c] = true
	}
	for _, s := range b.order {
		for _, z := range s.zones {
			if z.building != nil && z.building.ID != 0 {
				mark(z.building.Player, s.coord)
			}
			for _, u := range z.units {
				mark(u.Player, s.coord)
			}
		}
	}
	return spaces
}

// How many of each unit and building id a player was given, and the space of its village center (building id 1)
func (b *Board) Tally(player int) (units map[uint32]int, buildings map[uint32]int, home string) {
	units, buildings = map[uint32]int{}, map[uint32]int{}
	for _, s := range b.order {
		for _, z := range s.zones {
			if z.building != nil && z.building.Player == player {
				buildings[z.building.ID]++
				if z.building.ID == 1 {
					home = fmt.Sprintf("%d,%d", s.coord.X, s.coord.Y)
				}
			}
			for _, u := range z.units {
				if u.Player == player {
					units[u.ID]++
				}
			}
		}
	}
	return units, buildings, home
}

// Groups of passable spaces joined by borders, seams or links; 1 means every passable space is reachable
func (b *Board) PassableComponents() int {
	passable := map[*space]bool{}
	for _, s := range b.order {
		passable[s] = !s.Impassable()
	}
	seen, groups := map[*space]bool{}, 0
	for _, s := range b.order {
		if !passable[s] || seen[s] {
			continue
		}
		groups++
		for stack := []*space{s}; len(stack) > 0; {
			current := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if seen[current] {
				continue
			}
			seen[current] = true
			visit := func(n *space) {
				if passable[n] && !seen[n] {
					stack = append(stack, n)
				}
			}
			for _, n := range current.borders {
				visit(n)
			}
		}
	}
	return groups
}

func (b *Board) hopDistances(from *space) []uint16 {
	row := make([]uint16, len(b.order))
	for i := range row {
		row[i] = Unreachable
	}
	row[from.distanceIndex] = 0
	queue := make([]*space, 1, len(b.order))
	queue[0] = from
	for head := 0; head < len(queue); head++ {
		current := queue[head]
		visit := func(next *space) {
			if next != nil && row[next.distanceIndex] == Unreachable {
				row[next.distanceIndex] = row[current.distanceIndex] + 1
				queue = append(queue, next)
			}
		}
		for _, next := range current.borders {
			visit(next)
		}
	}
	return row
}

// Every non-border adjacency once, as "x,y>x,y:direction", sorted
func (b *Board) LinkLines() []string {
	lines := []string{}
	for _, s := range b.order {
		for d, v := range game_utils.AxialDirectionVectors() {
			if n := s.borders[d]; n != nil && s.key < n.key && (n.coord != *s.coord.Add(&v) || s.arrivals[d] != (d+3)%6) {
				lines = append(lines, fmt.Sprintf("%d,%d>%d,%d:%d", s.coord.X, s.coord.Y, n.coord.X, n.coord.Y, d))
			}
		}
	}
	slices.Sort(lines)
	return lines
}

// Each region as "name|bonus|sorted space keys", sorted
func (b *Board) RegionLines() []string {
	lines := []string{}
	for _, region := range b.Regions {
		keys := make([]uint, 0, len(region.Keys))
		for key := range region.Keys {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		lines = append(lines, fmt.Sprintf("%s|%v|%v", region.Name, region.GoldBonus, keys))
	}
	slices.Sort(lines)
	return lines
}
