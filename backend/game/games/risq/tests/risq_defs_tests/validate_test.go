package risq_defs_tests

import (
	"encoding/json"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"os"
	"testing"
)

func TestConfigUniqueIDs(t *testing.T) {
	defs.LoadConfig("../../config")
	check := func(file string, mapLen int) {
		data, _ := os.ReadFile("../../config/" + file)
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
	defs.LoadConfig("../../config")
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
