package mapgen

import "github.com/dgray001/gray_online/game/game_utils"

func floorDiv2(x int) int {
	if x >= 0 || x%2 == 0 {
		return x / 2
	}
	return x/2 - 1
}

func hexAreaForSize(size int) int {
	return 3*size*size + 3*size + 1
}

// Returns every space within hex distance radius of start (radius 0 -> just start, radius 1 -> start + 6 neighbors, etc.)
func hexRadiusSpaces(start Space, radius int) []Space {
	seen := map[uint]bool{start.Key(): true}
	result := []Space{start}
	frontier := []Space{start}
	for len(frontier) > 0 {
		cur := frontier[0]
		frontier = frontier[1:]
		for _, n := range cur.SortedAdjacent() {
			if seen[n.Key()] || int(game_utils.AxialDistance(start.Coordinate(), n.Coordinate())) > radius {
				continue
			}
			seen[n.Key()] = true
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
