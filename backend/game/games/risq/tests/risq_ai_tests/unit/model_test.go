package unit

import (
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"math/rand"
	"testing"
)

func TestParseModelEmpty(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	raw := map[string]any{"rules": []any{}}
	model := ai.ParseModel(raw, rng)
	if model == nil {
		t.Fatal("ParseModel returned nil")
	}
}
