package sims

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/config"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/aisim"
)

var mapFile string

func TestMain(m *testing.M) {
	cleanup := setupConfigDir()
	defer cleanup()
	os.Exit(m.Run())
}

func setupConfigDir() func() {
	dir, _ := os.MkdirTemp("", "sims-*")
	mapDir := filepath.Join(dir, "maps", "custom")
	_ = os.MkdirAll(mapDir, 0o755)
	mapFile = filepath.Join(mapDir, "ai-test.json")
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

const emptyModel = `{"rules":[]}`
const remote = `{"x":-3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":1},"units":[{"id":1,"player":1,"count":1}]}]}`

func once(actions string) string {
	return `{"rules":[{"when":{"turn_equals":{"amount":1}},"then":[` + actions + `]}]}`
}

func fixture(t *testing.T, bank, spaces, extra string, models ...string) *aisim.Game {
	t.Helper()
	document := fmt.Sprintf(`{"board_size":4,"players":%d,"space_gold_income":0,"starting_bank":%s,"spaces":[%s]%s}`, len(models), bank, spaces, extra)
	if err := os.WriteFile(mapFile, []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	return aisim.New(t, "custom:ai-test", models...)
}

func TestEmptyModelsResolveTurnsAndStop(t *testing.T) {
	g := fixture(t, `{}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":2,"player":0},"units":[{"id":1,"player":0,"count":1}]}]},`+remote, "", emptyModel, emptyModel)
	g.Run(t, 3)
	if len(g.Trace) != 6 {
		t.Fatalf("actions=%+v", g.Trace)
	}
	g.Stop(t)
	g.Stop(t)
}
