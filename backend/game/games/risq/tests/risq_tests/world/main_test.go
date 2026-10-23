package world

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/config"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

var (
	testConfig   = testconfig.Dir()
	customMapDir string
	baseDir      string
	pristine     *defs.ConfigSnapshot
)

func TestMain(m *testing.M) {
	cleanup := setupConfigDir()
	defer cleanup()
	os.Exit(m.Run())
}

func setupConfigDir() func() {
	dir, _ := os.MkdirTemp("", "world-*")
	baseDir = dir
	customMapDir = filepath.Join(dir, "maps", "custom")
	_ = os.MkdirAll(customMapDir, 0o755)
	copyFiles(dir)
	_ = defs.LoadConfig(dir)
	pristine = defs.SnapshotConfig()
	return func() { _ = defs.LoadConfig(testConfig); _ = os.RemoveAll(dir) }
}

func copyFiles(dir string) {
	files := []string{"ai/default.json", "bonuses.json", "buildings.json", "region_names.json", "resources.json", "techs.json", "terrains.json", "units.json"}
	for _, f := range files {
		data, _ := os.ReadFile(filepath.Join(testConfig, f))
		dst := filepath.Join(dir, f)
		_ = os.MkdirAll(filepath.Dir(dst), 0o755)
		_ = os.WriteFile(dst, data, 0o644)
	}
}

// A radius-4 two-slot custom map document; extra is more top-level fields, spaces the space entries
func mapDoc(extra string, spaces string) string {
	return `{"board_size":4,"players":2` + extra + `,"spaces":[` + spaces + `]}`
}

// A space entry with the given unit entries on its center zone
func spaceWith(x, y int, terrain int, units string) string {
	zones := ""
	if units != "" {
		zones = `,"zones":[{"x":0,"y":0,"units":[` + units + `]}]`
	}
	return fmt.Sprintf(`{"x":%d,"y":%d,"terrain":%d%s}`, x, y, terrain, zones)
}

func unit(id uint32, slot int) string {
	return fmt.Sprintf(`{"id":%d,"player":%d,"count":1}`, id, slot)
}

const (
	worker             = 11
	grunt              = 21
	longbow            = 32
	flatTerrain        = 1
	hillyTerrain       = 51
	mountainousTerrain = 101
	waterTerrain       = 152
)

// Starts a two-human game on the given map document, which exists only for this test
func startGame(t *testing.T, doc string) *harness.Game {
	t.Helper()
	t.Cleanup(func() { pristine.Restore(baseDir) })
	_ = os.WriteFile(filepath.Join(customMapDir, "world.json"), []byte(doc), 0o644)
	return harness.NewGame(t, "custom:world", 1, 2)
}

func writeCustomMap(name, doc string) {
	_ = os.WriteFile(filepath.Join(customMapDir, name+".json"), []byte(doc), 0o644)
}
