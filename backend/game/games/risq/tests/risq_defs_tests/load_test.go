package risq_defs_tests

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"os"
	"path/filepath"
	"testing"
)

func TestShippedConfigLoads(t *testing.T) {
	err := defs.LoadConfig("../../config")
	if err != nil {
		t.Fatalf("Failed to load shipped config: %v", err)
	}
	if len(defs.UnitConfigs) == 0 || len(defs.BuildingConfigs) == 0 {
		t.Errorf("Configs loaded but maps are empty")
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
