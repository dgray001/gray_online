package fakeboard

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/mapgen"
)

// Rows top to bottom and columns left to right, exactly like the engine, so seeded steps see the same order
func (b *Board) Allocate(board_size uint16) {
	b.size = board_size
	b.order, b.distances = nil, nil
	b.byCoord = map[game_utils.Coordinate2D]*space{}
	b.byKey = map[uint]*space{}
	n := int(board_size)
	for r := -n; r <= n; r++ {
		for q := max(-n, -n-r); q <= min(n, n-r); q++ {
			s := newSpace(b, game_utils.Coordinate2D{X: q, Y: r})
			b.order = append(b.order, s)
			b.byCoord[s.coord] = s
			b.byKey[s.key] = s
		}
	}
	for _, s := range b.order {
		for d, v := range game_utils.AxialDirectionVectors() {
			s.borders[d] = b.byCoord[*s.coord.Add(&v)]
		}
	}
}

func (b *Board) RemoveSpace(removed mapgen.Space) {
	s := removed.(*space)
	b.distances = nil
	for d, neighbor := range s.borders {
		if neighbor != nil {
			neighbor.borders[(d+3)%6] = nil
		}
	}
	delete(b.byCoord, s.coord)
	delete(b.byKey, s.key)
	for i, other := range b.order {
		if other == s {
			b.order = append(b.order[:i], b.order[i+1:]...)
			return
		}
	}
}
