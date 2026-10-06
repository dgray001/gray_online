package fakeboard

import (
	"fmt"

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
			for _, n := range current.links {
				visit(n)
			}
		}
	}
	return groups
}
