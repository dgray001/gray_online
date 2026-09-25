package risq

import (
	"math"

	"github.com/dgray001/gray_online/game/game_utils"
)

func floorDiv2(x int) int {
	if x >= 0 || x%2 == 0 {
		return x / 2
	}
	return x/2 - 1
}

func hexAreaForSize(size int) int {
	return 3*size*size + 3*size + 1
}

func cubeRound(x float64, y float64, z float64) (int, int, int) {
	rx, ry, rz := math.Round(x), math.Round(y), math.Round(z)
	dx, dy, dz := math.Abs(rx-x), math.Abs(ry-y), math.Abs(rz-z)
	if dx > dy && dx > dz {
		rx = -ry - rz
	} else if dy > dz {
		ry = -rx - rz
	} else {
		rz = -rx - ry
	}
	return int(rx), int(ry), int(rz)
}

func hexLine(from game_utils.Coordinate2D, to game_utils.Coordinate2D, risq *GameRisq) []*RisqSpace {
	dist := int(game_utils.AxialDistance(from, to))
	if dist == 0 {
		if space := risq.getSpace(&from); space != nil {
			return []*RisqSpace{space}
		}
		return nil
	}
	x1, z1 := float64(from.X), float64(from.Y)
	y1 := -x1 - z1
	x2, z2 := float64(to.X), float64(to.Y)
	y2 := -x2 - z2
	spaces := make([]*RisqSpace, 0, dist+1)
	seen := make(map[uint]bool)
	for i := 0; i <= dist; i++ {
		t := float64(i) / float64(dist)
		rx, _, rz := cubeRound(x1+(x2-x1)*t, y1+(y2-y1)*t, z1+(z2-z1)*t)
		c := game_utils.Coordinate2D{X: rx, Y: rz}
		space := risq.getSpace(&c)
		if space == nil || seen[space.coordinate_key] {
			continue
		}
		seen[space.coordinate_key] = true
		spaces = append(spaces, space)
	}
	return spaces
}

// Returns every space within hex distance radius of start (radius 0 -> just start, radius 1 -> start + 6 neighbors, etc.)
func hexRadiusSpaces(start *RisqSpace, radius int) []*RisqSpace {
	seen := map[uint]bool{start.coordinate_key: true}
	result := []*RisqSpace{start}
	frontier := []*RisqSpace{start}
	for len(frontier) > 0 {
		cur := frontier[0]
		frontier = frontier[1:]
		for _, n := range cur.sortedAdjacentSpaces() {
			if seen[n.coordinate_key] || int(game_utils.AxialDistance(start.coordinate, n.coordinate)) > radius {
				continue
			}
			seen[n.coordinate_key] = true
			result = append(result, n)
			frontier = append(frontier, n)
		}
	}
	return result
}

// Rotates an axial coordinate by steps increments of sixty degrees
func rotateAxial(c game_utils.Coordinate2D, steps int) game_utils.Coordinate2D {
	x, z := c.X, c.Y
	y := -x - z
	n := ((steps % 6) + 6) % 6
	for range n {
		x, y, z = -y, -z, -x
	}
	return game_utils.Coordinate2D{X: x, Y: z}
}

func hexRingSectors(center game_utils.Coordinate2D, radius int) [6][]game_utils.Coordinate2D {
	var sectors [6][]game_utils.Coordinate2D
	if radius <= 0 {
		return sectors
	}
	dirs := game_utils.AxialDirectionVectors()
	hex := game_utils.Coordinate2D{X: center.X + dirs[4].X*radius, Y: center.Y + dirs[4].Y*radius}
	for i := 0; i < 6; i++ {
		dir := dirs[i]
		for j := 0; j < radius; j++ {
			sectors[i] = append(sectors[i], hex)
			hex = game_utils.Coordinate2D{X: hex.X + dir.X, Y: hex.Y + dir.Y}
		}
	}
	return sectors
}
