package world

import (
	"fmt"
	"strings"
	"testing"
)

// Every space within radius 2 of the origin; P0's villager stands on the given zone of (0,0) and P1's is far in the corner
func zoneVisionLevels(t *testing.T, zx, zy int) map[[2]int]int {
	t.Helper()
	var spaces []string
	for r := -2; r <= 2; r++ {
		for q := max(-2, -2-r); q <= min(2, 2-r); q++ {
			switch {
			case q == 0 && r == 0:
				spaces = append(spaces, fmt.Sprintf(`{"x":0,"y":0,"terrain":1,"zones":[{"x":%d,"y":%d,"units":[%s]}]}`, zx, zy, unit(villager, 0)))
			case q == 2 && r == -2:
				spaces = append(spaces, spaceWith(q, r, grass, unit(villager, 1)))
			default:
				spaces = append(spaces, spaceWith(q, r, grass, ""))
			}
		}
	}
	g := startGame(t, mapDoc("", strings.Join(spaces, ",")))
	state := g.State(g.Human(0))
	levels := map[[2]int]int{}
	for _, row := range state.Spaces {
		for _, space := range row {
			if space != nil {
				levels[[2]int{space.Coordinate.X, space.Coordinate.Y}] = space.Visibility
			}
		}
	}
	return levels
}

func TestVisionFromTheCenterZoneSeesAllNeighborsPoorly(t *testing.T) {
	levels := zoneVisionLevels(t, 0, 0)
	for _, neighbor := range [][2]int{{1, 0}, {1, -1}, {0, -1}, {-1, 0}, {-1, 1}, {0, 1}} {
		if levels[neighbor] != poor {
			t.Errorf("neighbor %v visibility %d from the center zone, want poor", neighbor, levels[neighbor])
		}
	}
}

func TestEveryEdgeZoneSeesItsFacingNeighborClearly(t *testing.T) {
	for _, d := range [][2]int{{1, -1}, {0, -1}, {-1, 0}, {-1, 1}, {0, 1}} {
		levels := zoneVisionLevels(t, d[0], d[1])
		if levels[d] != good {
			t.Errorf("edge zone %v: facing neighbor visibility %d, want good", d, levels[d])
		}
		if levels[[2]int{-2 * d[0], -2 * d[1]}] != unexplored {
			t.Errorf("edge zone %v: the space two steps behind is visible, want unexplored", d)
		}
	}
}

func TestVisionFromAnEdgeZoneSeesItsFacingNeighborClearly(t *testing.T) {
	levels := zoneVisionLevels(t, 1, 0)
	if levels[[2]int{0, 0}] != good || levels[[2]int{1, 0}] != good {
		t.Errorf("own space %d and the neighbor the edge faces %d, want both good", levels[[2]int{0, 0}], levels[[2]int{1, 0}])
	}
	for _, neighbor := range [][2]int{{1, -1}, {0, 1}, {0, -1}, {-1, 0}, {-1, 1}} {
		if levels[neighbor] != poor {
			t.Errorf("neighbor %v visibility %d from an east edge zone, want poor", neighbor, levels[neighbor])
		}
	}
	if levels[[2]int{-2, 0}] != unexplored {
		t.Errorf("the far side %d, want unexplored", levels[[2]int{-2, 0}])
	}
}
