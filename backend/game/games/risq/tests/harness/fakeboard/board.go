// Package fakeboard is an in-memory mapgen.Board that mirrors the engine's ordering and adjacency rules.
package fakeboard

import (
	"math/rand"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/internal/mapgen"
)

type Board struct {
	size       uint16
	order      []*space
	byCoord    map[game_utils.Coordinate2D]*space
	byKey      map[uint]*space
	distances  map[*space][]uint16
	Regions    []Region
	Banks      map[int]mapgen.StartingBank
	Techs      map[int][]uint32
	Unlimited  bool
	GoldIncome *float64
	NeedRegion *bool
	Violations []string
}

type Region struct {
	Name      string
	GoldBonus float64
	Keys      map[uint]bool
}

func New() *Board {
	return &Board{Banks: map[int]mapgen.StartingBank{}, Techs: map[int][]uint32{}}
}

func (b *Board) BoardSize() uint16 {
	return b.size
}

func (b *Board) Spaces() []mapgen.Space {
	spaces := make([]mapgen.Space, len(b.order))
	for i, s := range b.order {
		spaces[i] = s
	}
	return spaces
}

func (b *Board) Space(c game_utils.Coordinate2D) mapgen.Space {
	if s, ok := b.byCoord[c]; ok {
		return s
	}
	return nil
}

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
			neighbor.borders[s.arrivals[d]], neighbor.sorted = nil, nil
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

// Runs a "script:<name>" or "custom:<name>" map the way the engine does: one rng seeded from the game seed
func Generate(map_name string, num_players int, seed int64) (*Board, error) {
	return GenerateWithSize(map_name, num_players, defs.DefaultMapSize(num_players), seed)
}

func GenerateWithSize(map_name string, num_players int, map_size defs.MapSize, seed int64) (*Board, error) {
	board := New()
	settings := mapgen.ResolveSettings(map[string]any{"map_size": float64(map_size)}, num_players)
	err := mapgen.Generate(board, rand.New(rand.NewSource(seed)), num_players, settings, map_name)
	return board, err
}

// Runs an in-memory custom map document
func GenerateDocument(doc string, num_players int) (*Board, error) {
	board := New()
	err := mapgen.GenerateCustomDocument(board, num_players, []byte(doc))
	return board, err
}
