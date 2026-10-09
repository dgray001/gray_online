package risq_defs_tests

import (
	"encoding/json"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/config"
	"os"
	"testing"
)

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
		if u.Unit_type <= 0 || u.Unit_type > defs.UnitType_CAVALRY {
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
