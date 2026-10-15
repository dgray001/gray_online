package risq_defs_tests

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/config"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestSharedTestConfigLoads(t *testing.T) {
	err := testconfig.Load()
	if err != nil {
		t.Fatalf("Failed to load shared test config: %v", err)
	}
	if len(defs.UnitConfigs) == 0 || len(defs.BuildingConfigs) == 0 {
		t.Errorf("Configs loaded but maps are empty")
	}
}

func TestConfigDirReplacesOnlyItsFilesAndLeavesDefaultsAlone(t *testing.T) {
	if err := testconfig.Load(); err != nil {
		t.Fatal(err)
	}
	config, err := defs.LoadConfigDir("override_units")
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Units) != 1 || config.Units[5].Display_name != "Solo" {
		t.Errorf("override units not used: %v", config.Units)
	}
	if len(config.Buildings) != len(defs.BuildingConfigs) || len(config.Techs) != len(defs.TechConfigs) {
		t.Error("files missing from the override folder did not fall back to the defaults")
	}
	if _, ok := defs.UnitConfigs[5]; ok || len(defs.UnitConfigs) < 2 {
		t.Error("loading an override changed the default tables")
	}
}

func TestConfigDirRejectsBadNamesAndMalformedFiles(t *testing.T) {
	if err := testconfig.Load(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", "../escape", "Upper", "a/b"} {
		if _, err := defs.LoadConfigDir(name); err == nil {
			t.Errorf("config dir %q was accepted", name)
		}
	}
	if _, err := defs.LoadConfigDir("override_bad"); err == nil {
		t.Error("malformed override file did not return an error")
	}
}

func TestMercenaryMarkupDeltasFromBonusesAndTechsStack(t *testing.T) {
	if err := testconfig.Load(); err != nil {
		t.Fatal(err)
	}
	config, err := defs.LoadConfigDir("override_markup")
	if err != nil {
		t.Fatal(err)
	}
	if got := config.MercenaryCostMarkupDelta(nil); math.Abs(got-(-0.3)) > 1e-9 {
		t.Errorf("bonus deltas summed to %v, want -0.3", got)
	}
	if got := config.MercenaryCostMarkupDelta([]uint32{7, 999}); math.Abs(got-(-0.25)) > 1e-9 {
		t.Errorf("bonus plus tech deltas summed to %v, want -0.25", got)
	}
}

func TestMalformedConfigPanics(t *testing.T) {
	tmp := t.TempDir()
	files := []string{"bonuses.json", "buildings.json", "region_names.json", "resources.json", "techs.json", "terrains.json", "units.json"}
	for _, file := range files {
		os.WriteFile(filepath.Join(tmp, file), []byte("[]"), 0644)
	}
	os.WriteFile(filepath.Join(tmp, "units.json"), []byte("{bad json}"), 0644)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic loading malformed config")
		}
	}()
	// Ignore normal error, we are testing for a panic during parsing
	_ = defs.LoadConfig(tmp)
}
