package unit

import (
	"encoding/json"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/game/games/risq/tests/config"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTestConfigs(t *testing.T) {
	configsDir := filepath.Join(testconfig.Dir(), "ai")
	entries, err := os.ReadDir(configsDir)
	if err != nil {
		t.Fatalf("failed to read config directory: %v", err)
	}
	loaded := 0

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		loaded++

		t.Run(entry.Name(), func(t *testing.T) {
			path := filepath.Join(configsDir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read config %s: %v", path, err)
			}

			var raw map[string]any
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("failed to unmarshal config %s: %v", path, err)
			}

			rng := rand.New(rand.NewSource(1))
			model := ai.ParseModelWithDiagnostics(raw, rng, io.Discard)
			if _, ok := model.(*ai.RulesModel); !ok {
				t.Fatalf("config %s parsed to %T, expected *ai.RulesModel", entry.Name(), model)
			}

			for _, view := range []*fakeView{{}, {units: []ai.UnitView{{InternalID: 73, UnitID: 13, Kind: ai.UnitMilitary, Type: ai.UnitTypeInfantry}}}} {
				for turn := 0; turn < 2; turn++ {
					model.DecideOrders(view)
				}
			}
		})
	}
	if loaded == 0 {
		t.Fatal("no test-owned AI configs")
	}
}
