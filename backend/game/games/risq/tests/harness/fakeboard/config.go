package fakeboard

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

var sharedConfigFiles = []string{"bonuses.json", "buildings.json", "region_names.json", "resources.json", "techs.json", "terrains.json", "units.json"}

// Swaps the engine to a temp copy of the shipped config plus the given maps (name to JSON), restored on cleanup.
func UseConfig(t *testing.T, shipped string, scripts map[string]string, customs map[string]string) {
	t.Helper()
	dir := t.TempDir()
	for _, file := range append([]string{"ai/default.json"}, sharedConfigFiles...) {
		data, err := os.ReadFile(filepath.Join(shipped, file))
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(dir, file), data)
	}
	for name, doc := range scripts {
		writeFile(t, filepath.Join(dir, "maps", "scripted", name+".json"), []byte(doc))
	}
	for name, doc := range customs {
		writeFile(t, filepath.Join(dir, "maps", "custom", name+".json"), []byte(doc))
	}
	if err := defs.LoadConfig(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := defs.LoadConfig(shipped); err != nil {
			t.Errorf("restoring the shipped config: %v", err)
		}
	})
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
