package fakeboard

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

var (
	pristineMu sync.Mutex
	pristine   = map[string]*defs.ConfigSnapshot{}
	files      = map[string]map[string][]byte{}
)

var sharedConfigFiles = []string{"ai/default.json", "bonuses.json", "buildings.json", "region_names.json", "resources.json", "techs.json", "terrains.json", "units.json"}

// The parsed config of config_dir, read from disk once per process
func pristineConfig(t *testing.T, config_dir string) (*defs.ConfigSnapshot, map[string][]byte) {
	t.Helper()
	pristineMu.Lock()
	defer pristineMu.Unlock()
	if snapshot, ok := pristine[config_dir]; ok {
		return snapshot, files[config_dir]
	}
	if err := defs.LoadConfig(config_dir); err != nil {
		t.Fatal(err)
	}
	contents := map[string][]byte{}
	for _, file := range sharedConfigFiles {
		data, err := os.ReadFile(filepath.Join(config_dir, file))
		if err != nil {
			t.Fatal(err)
		}
		contents[file] = data
	}
	pristine[config_dir], files[config_dir] = defs.SnapshotConfig(), contents
	return pristine[config_dir], contents
}

// Points the engine at a temporary directory holding the fixture maps over a pristine copy of the test config, restoring it on cleanup.
func UseConfig(t *testing.T, config_dir string, scripts map[string]string, customs map[string]string) string {
	t.Helper()
	snapshot, shared := pristineConfig(t, config_dir)
	dir := t.TempDir()
	for file, data := range shared {
		writeFile(t, filepath.Join(dir, file), data)
	}
	for name, doc := range scripts {
		writeFile(t, filepath.Join(dir, "maps", "scripted", name+".json"), []byte(doc))
	}
	for name, doc := range customs {
		writeFile(t, filepath.Join(dir, "maps", "custom", name+".json"), []byte(doc))
	}
	snapshot.Restore(dir)
	t.Cleanup(func() { snapshot.Restore(config_dir) })
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
