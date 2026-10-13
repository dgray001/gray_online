package bridge

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/config"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

var mapFile string

func TestMain(m *testing.M) {
	cleanup := setupConfigDir()
	defer cleanup()
	os.Exit(m.Run())
}

func setupConfigDir() func() {
	dir, err := os.MkdirTemp("", "bridge-*")
	if err != nil {
		panic(err)
	}
	mapDir := filepath.Join(dir, "maps", "custom")
	_ = os.MkdirAll(mapDir, 0o755)
	mapFile = filepath.Join(mapDir, "bridge-test.json")
	copyFiles(dir)
	_ = defs.LoadConfig(dir)
	return func() { _ = defs.LoadConfig(testconfig.Dir()); _ = os.RemoveAll(dir) }
}

func copyFiles(dir string) {
	files := []string{"ai/default.json", "bonuses.json", "buildings.json", "region_names.json", "resources.json", "techs.json", "terrains.json", "units.json"}
	for _, f := range files {
		data, _ := os.ReadFile(filepath.Join(testconfig.Dir(), f))
		dst := filepath.Join(dir, f)
		_ = os.MkdirAll(filepath.Dir(dst), 0o755)
		_ = os.WriteFile(dst, data, 0o644)
	}
}

func bridgeGame(t *testing.T, spaces string, configure ...func()) (*harness.Game, *game.Player) {
	t.Helper()
	document := `{"board_size":4,"players":2,"spaces":[` + spaces + `]}`
	if err := os.WriteFile(mapFile, []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	if len(configure) > 0 {
		saved := defs.UnitConfigs[11]
		t.Cleanup(func() { defs.UnitConfigs[11] = saved })
		for _, apply := range configure {
			apply()
		}
	}
	g := harness.NewGame(t, "custom:bridge-test", 1, 2)
	player := game.CreateAiPlayer("probe", g.Base)
	player.Player_id = 0
	return g, player
}
