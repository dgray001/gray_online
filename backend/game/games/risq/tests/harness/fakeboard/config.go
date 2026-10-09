package fakeboard

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

var sharedConfigFiles = []string{"bonuses.json", "buildings.json", "region_names.json", "resources.json", "techs.json", "terrains.json", "units.json"}

// Loads a temporary copy of the test config with fixture maps, restoring the source on cleanup.
func UseConfig(t *testing.T, config_dir string, scripts map[string]string, customs map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for _, file := range append([]string{"ai/default.json"}, sharedConfigFiles...) {
		data, err := os.ReadFile(filepath.Join(config_dir, file))
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
		if err := defs.LoadConfig(config_dir); err != nil {
			t.Errorf("restoring the test config: %v", err)
		}
	})
	return dir
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
