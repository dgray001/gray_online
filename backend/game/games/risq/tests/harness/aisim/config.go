package aisim

import (
	"encoding/json"
	"math/rand"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

func WriteModel(t *testing.T, name, document string) {
	t.Helper()
	var raw map[string]any
	if err := json.Unmarshal([]byte(document), &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := ai.ParseModel(raw, rand.New(rand.NewSource(1))).(*ai.RulesModel); !ok {
		t.Fatalf("invalid model %s", name)
	}
	if err := defs.WriteConfigFile([]byte(document), "ai", name+".json"); err != nil {
		t.Fatal(err)
	}
}
