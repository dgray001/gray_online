package unit

import (
	"math/rand"
	"testing"
	"github.com/dgray001/gray_online/game/games/risq/ai"
)

func TestParseModelInvalidRules(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	raw := map[string]any{"rules": "not a list"}
	model := ai.ParseModel(raw, rng)
	if _, ok := model.(ai.NoopModel); !ok {
		t.Fatalf("expected NoopModel for invalid rules, got %T", model)
	}
}
