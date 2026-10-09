package risq_defs_tests

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/config"
	"testing"
)

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
