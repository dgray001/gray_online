package combat

import (
	"math"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestHealthNetting(t *testing.T) {
	g := combatGame(t, 13, 13)
	p0, p1 := g.Human(0), g.Human(1)
	u0 := g.Self(p0).Units[0]
	u1 := g.Self(p1).Units[0]
	g.Submit(p0, harness.OrderAttackUnit([]uint64{u0.InternalID}, u1.InternalID))
	g.Submit(p1, harness.OrderAttackUnit([]uint64{u1.InternalID}, u0.InternalID))
	after0 := g.Self(p0).Units[0]
	after1 := g.Self(p1).Units[0]
	damage0 := u0.CombatStats.Health - after0.CombatStats.Health
	damage1 := u1.CombatStats.Health - after1.CombatStats.Health
	if damage0 <= 0 || math.Abs(damage0-damage1) > 1e-6 {
		t.Errorf("expected equal simultaneous damage, got p0: %v, p1: %v", damage0, damage1)
	}
}

func TestTimeToKill(t *testing.T) {
	if u := risq.ComputeUnitVsUnitTTK(11, 11); !u.Possible || u.Turns != 2 || u.Ticks != 7 {
		t.Fatalf("unexpected unit TTK: %+v", u)
	}
	if b := risq.ComputeUnitVsBuildingTTK(11, 2); !b.Possible || b.Ticks <= 0 {
		t.Fatalf("unexpected building TTK: %+v", b)
	}
	if ut := risq.ComputeUnitVsUnitTTKWithTechs(11, 11, []uint32{2}, []uint32{3}); !ut.Possible {
		t.Fatalf("unexpected tech TTK: %+v", ut)
	}
	if bt := risq.ComputeUnitVsBuildingTTKWithTechs(11, 2, []uint32{2}); !bt.Possible {
		t.Fatalf("unexpected building tech TTK: %+v", bt)
	}
	if len(risq.AllUnitIDs()) == 0 || len(risq.AllBuildingIDs()) == 0 {
		t.Fatal("empty entity ID lists")
	}
	if risq.UnitName(1) != "Villager" || risq.BuildingName(1) != "Village Center" {
		t.Fatalf("unexpected display names: %q, %q", risq.UnitName(1), risq.BuildingName(1))
	}
}

func TestAttackTypesAndPenetration(t *testing.T) {
	g := combatGame(t, 12, 13) // 12: 10 piercing att. 13: 3 piercing def, 50 hp
	p0, p1 := g.Human(0), g.Human(1)
	u0 := g.Self(p0).Units[0]
	u1 := g.Self(p1).Units[0]
	g.Submit(p0, harness.OrderAttackUnit([]uint64{u0.InternalID}, u1.InternalID))
	g.Submit(p1) // do nothing
	after1 := g.Self(p1).Units[0]
	damage := u1.CombatStats.Health - after1.CombatStats.Health
	// ratio 3/10 is under the knee: (10-3) * stamina / 10
	if want := 7 * float64(u0.CurrentStamina) / 10; math.Abs(damage-want) > 1e-6 {
		t.Errorf("expected damage %v, got %v", want, damage)
	}
}

func TestPenetrationReducesDefense(t *testing.T) {
	g := combatGame(t, 13, 13) // 18 attack, 5% penetration vs 4 blunt + 3 piercing defense
	p0, p1 := g.Human(0), g.Human(1)
	u0 := g.Self(p0).Units[0]
	u1 := g.Self(p1).Units[0]
	g.Submit(p0, harness.OrderAttackUnit([]uint64{u0.InternalID}, u1.InternalID))
	g.Submit(p1)
	damage := u1.CombatStats.Health - g.Self(p1).Units[0].CombatStats.Health
	defense := 4*0.95 + 3*0.95
	want := (18 - defense) * float64(u0.CurrentStamina) / 10
	if math.Abs(damage-want) > 1e-6 {
		t.Errorf("expected damage %v, got %v", want, damage)
	}
}
