package unit

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"github.com/dgray001/gray_online/game/games/risq/ai"
)

func TestParseShippedConfigs(t *testing.T) {
	configsDir := "../../config/ai"
	entries, err := os.ReadDir(configsDir)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("Config directory %s not found", configsDir)
		}
		t.Fatalf("failed to read config directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

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
			model := ai.ParseModel(raw, rng)
			if _, ok := model.(*ai.RulesModel); !ok {
				t.Fatalf("config %s parsed to %T, expected *ai.RulesModel", entry.Name(), model)
			}

			// Optional: test DecideOrders on empty view to make sure it doesn't panic
			view := &fakeView{}
			model.DecideOrders(view)
		})
	}
}
