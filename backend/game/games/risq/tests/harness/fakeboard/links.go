package fakeboard

import (
	"fmt"

	"github.com/dgray001/gray_online/game/games/risq/internal/mapgen"
)

func (b *Board) ConnectSpaces(a, c mapgen.Space, direction int, to_direction int) error {
	first, second := a.(*space), c.(*space)
	b.distances = nil
	if direction < 0 || direction > 5 || to_direction < 0 || to_direction > 5 {
		return fmt.Errorf("seam directions %d and %d are not 0 to 5", direction, to_direction)
	}
	if first.borders[direction] != nil || second.borders[to_direction] != nil {
		return fmt.Errorf("seam edge between %v and %v is already connected", first.coord, second.coord)
	}
	first.borders[direction], second.borders[to_direction] = second, first
	first.arrivals[direction], second.arrivals[to_direction] = to_direction, direction
	first.sorted, second.sorted = nil, nil
	return nil
}

// The distance the engine reports between spaces with no path
const Unreachable = 1 << 15

// Steps between two spaces over borders, seams and links; Unreachable when there is no path
func (b *Board) Distance(a, c mapgen.Space) int {
	from, to := a.(*space), c.(*space)
	if from == to {
		return 0
	}
	if b.distances == nil {
		b.distances = map[*space][]uint16{}
		for i, s := range b.order {
			s.distanceIndex = i
		}
	}
	if b.distances[from] == nil {
		b.distances[from] = b.hopDistances(from)
	}
	return int(b.distances[from][to.distanceIndex])
}
