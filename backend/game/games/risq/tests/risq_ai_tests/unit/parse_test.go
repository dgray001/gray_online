package unit

import (
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"math/rand"
	"testing"
)

func TestParseModelInvalidRules(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	raw := map[string]any{"rules": "not a list"}
	model := ai.ParseModel(raw, rng)
	if _, ok := model.(ai.NoopModel); !ok {
		t.Fatalf("expected NoopModel for invalid rules, got %T", model)
	}
}
