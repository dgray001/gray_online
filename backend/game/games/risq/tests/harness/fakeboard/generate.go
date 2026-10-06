package fakeboard

import (
	"math/rand"

	"github.com/dgray001/gray_online/game/games/risq/internal/mapgen"
)

// Runs a "script:<name>" or "custom:<name>" map the way the engine does: one rng seeded from the game seed
func Generate(map_name string, num_players int, seed int64) (*Board, error) {
	board := New()
	err := mapgen.Generate(board, rand.New(rand.NewSource(seed)), num_players, map_name)
	return board, err
}

// Runs an in-memory custom map document
func GenerateDocument(doc string, num_players int) (*Board, error) {
	board := New()
	err := mapgen.GenerateCustomDocument(board, num_players, []byte(doc))
	return board, err
}
