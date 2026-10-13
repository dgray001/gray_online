package shape

import (
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/config"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

var (
	testConfig = testconfig.Dir()
	pristine   *defs.ConfigSnapshot
)

const targetSpace = `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":1},"units":[{"id":1,"player":1,"count":1}]},{"x":1,"y":0,"terrain_override":24,"resource":41},{"x":-1,"y":0,"building":{"id":3,"player":1}},{"x":0,"y":-1,"units":[{"id":11,"player":1,"count":1}]}]}`
const scoutSpaces = `,{"x":-1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":15,"player":0,"count":1}]}]},{"x":-2,"y":0,"terrain":1},{"x":-3,"y":0,"terrain":1},{"x":-4,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":15,"player":0,"count":1}]}]}`
const hiddenSpaces = `,{"x":1,"y":0,"terrain":1},{"x":2,"y":0,"terrain":1},{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":1},"units":[{"id":1,"player":1,"count":1}]},{"x":1,"y":0,"resource":11}]},{"x":4,"y":0,"terrain":1}`
const regions = `,"regions":[{"name":"Border","gold_bonus":7,"spaces":[[0,0],[3,0]]},{"name":"Secret","gold_bonus":9,"spaces":[[4,0]]}]`

func TestMain(m *testing.M) {
	cleanup := setupConfigDir()
	defer cleanup()
	os.Exit(m.Run())
}

func setupConfigDir() func() {
	dir, _ := os.MkdirTemp("", "shape-*")
	mapDir := filepath.Join(dir, "maps", "custom")
	_ = os.MkdirAll(mapDir, 0o755)
	doc := `{"board_size":4,"players":2,"unlimited_population":true,"starting_techs":[4],"starting_bank":{"food":1000,"wood":1000,"stone":1000,"gold":1000},"spaces":[` + targetSpace + scoutSpaces + hiddenSpaces + `]` + regions + `}`
	_ = os.WriteFile(filepath.Join(mapDir, "shape.json"), []byte(doc), 0o644)
	metaDoc := `{"board_size":4,"players":2,"spaces":[` + targetSpace + scoutSpaces + hiddenSpaces + `],"background_image":{"name":"map.png","top_left":[-4,-2],"top_right":[4,2]},"connections":[{"from":[-4,0],"to":[4,0],"direction":3}]}`
	_ = os.WriteFile(filepath.Join(mapDir, "metadata.json"), []byte(metaDoc), 0o644)
	copyFiles(dir)
	_ = defs.LoadConfig(dir)
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

func snapshotConfigs() func() {
	units := maps.Clone(defs.UnitConfigs)
	buildings := maps.Clone(defs.BuildingConfigs)
	techs := maps.Clone(defs.TechConfigs)
	return func() {
		defs.UnitConfigs = units
		defs.BuildingConfigs = buildings
		defs.TechConfigs = techs
	}
}

func shapeGame(t *testing.T, level uint8) *harness.Game {
	t.Helper()
	return configuredGame(t, level, func() {})
}

func configuredGame(t *testing.T, level uint8, configure func()) *harness.Game {
	t.Helper()
	t.Cleanup(snapshotConfigs())
	scout := defs.UnitConfigs[1]
	scout.Vision = defs.RisqVision{Space: max(level, defs.VisibilityGood), Edge_adjacent: level, Adjacent: level, Edge_opposite: level}
	defs.UnitConfigs[15] = scout
	villager := defs.UnitConfigs[1]
	villager.Turn_stamina = 1
	defs.UnitConfigs[1] = villager
	configure()
	return harness.NewGame(t, "custom:shape", 1, 2)
}
