package risq_defs_tests

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/config"
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

func TestCostComputePoints(t *testing.T) {
	c := defs.RisqResourceCost{Food: 10, Wood: 20, Stone: 30, Gold: 40}
	if pts := c.Points(); pts != 140 {
		t.Errorf("Expected 140 points, got %d", pts)
	}
	scaled := c.Scale(2.0)
	if scaled.Food != 20 || scaled.Gold != 80 {
		t.Errorf("Scale failed: %+v", scaled)
	}
}

func TestSumTargetedBonus(t *testing.T) {
	testconfig.Load()
	orig := defs.TechConfigs
	defer func() { defs.TechConfigs = orig }()

	defs.TechConfigs = map[uint32]defs.TechConfig{
		999: {
			Bonus:              defs.CombatBonus{Attack_blunt: 5},
			Affects_unit_types: []defs.UnitType{defs.UnitType_INFANTRY},
			Target_types:       []defs.TargetType{{Orderable_type: defs.OrderableType_UNIT}},
		},
	}
	bonus := defs.SumTargetedBonus(1, defs.UnitType_INFANTRY, []uint32{999}, 2, defs.UnitType_INFANTRY, defs.OrderableType_UNIT)
	if bonus.Attack_blunt != 5 {
		t.Errorf("Expected 5 damage bonus, got %v", bonus.Attack_blunt)
	}
}

func TestTechTargeting(t *testing.T) {
	tech := defs.TechConfig{
		Affects_unit_types: []defs.UnitType{defs.UnitType_INFANTRY},
		Affects_unit_ids:   []uint32{10},
		Target_types:       []defs.TargetType{{Orderable_type: defs.OrderableType_UNIT, Object_type: int(defs.UnitType_CAVALRY)}},
	}
	if !tech.AppliesTo(99, defs.UnitType_INFANTRY) {
		t.Errorf("Should apply by type")
	}
	if !tech.AppliesTo(10, defs.UnitType_ECONOMIC) {
		t.Errorf("Should apply by id")
	}
	if tech.AppliesTo(99, defs.UnitType_ECONOMIC) {
		t.Errorf("Should not apply")
	}

	if !tech.MatchesTarget(50, defs.UnitType_CAVALRY, defs.OrderableType_UNIT) {
		t.Errorf("Should match target")
	}
	if tech.MatchesTarget(50, defs.UnitType_INFANTRY, defs.OrderableType_UNIT) {
		t.Errorf("Should not match target")
	}
}

func TestConfigUniqueIDs(t *testing.T) {
	testconfig.Load()
	check := func(file string, mapLen int) {
		data, _ := os.ReadFile(testconfig.Dir() + "/" + file)
		var entries []interface{}
		json.Unmarshal(data, &entries)
		if len(entries) != mapLen {
			t.Errorf("Duplicate IDs in %s: %d entries but %d in map", file, len(entries), mapLen)
		}
	}
	check("units.json", len(defs.UnitConfigs))
	check("buildings.json", len(defs.BuildingConfigs))
	check("techs.json", len(defs.TechConfigs))
}

func TestEnumValidity(t *testing.T) {
	testconfig.Load()
	for id, u := range defs.UnitConfigs {
		if u.Attack_type != defs.AttackType_NONE && (u.Attack_type <= 0 || u.Attack_type >= defs.AttackType_BLUNT_PIERCING_MAGIC+1) {
			t.Errorf("Invalid AttackType %v on %d", u.Attack_type, id)
		}
		if u.Unit_type <= 0 || u.Unit_type > defs.UnitType_SIEGE {
			t.Errorf("Bad unit type on %d", id)
		}
	}
	for id, b := range defs.BuildingConfigs {
		if b.Attack_type != defs.AttackType_NONE && (b.Attack_type <= 0 || b.Attack_type >= defs.AttackType_BLUNT_PIERCING_MAGIC+1) {
			t.Errorf("Invalid AttackType on building %d", id)
		}
	}
}

func TestMapSizeDefaults(t *testing.T) {
	want := []defs.MapSize{1, 1, 2, 3, 4, 5, 6, 7, 8, 8, 9, 9}
	for i, size := range want {
		for _, raw := range []any{nil, float64(0), float64(-1), float64(10), float64(256), float64(1.5), "5"} {
			if got := defs.ResolveMapSize(raw, i+1); got != size {
				t.Errorf("%d players, setting %v: size %d, want %d", i+1, raw, got, size)
			}
		}
	}
}

func TestMapSizeOverrides(t *testing.T) {
	for size := defs.MapSize_MINUSCULE; size <= defs.MapSize_GIGANTIC; size++ {
		if got := defs.ResolveMapSize(float64(size), 2); got != size {
			t.Errorf("setting %d: size %d", size, got)
		}
	}
}
