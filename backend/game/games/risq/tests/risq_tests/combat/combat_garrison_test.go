package combat

import (
	"math"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestTickHistoryGarrisonAttack(t *testing.T) {
	t.Run("own-action-preempts-garrison-attack", func(t *testing.T) {
		g := garrisonCombatGame(t)
		p, enemy := g.Human(0), g.Human(1)
		u, target := g.Self(p).Units[0], g.Self(enemy).Units[0]
		b := g.Self(p).Buildings[0]
		passive(g, p, u.InternalID)
		passive(g, enemy, target.InternalID)
		g.Submit(p, harness.OrderGarrison([]uint64{u.InternalID}, b.InternalID))
		g.EndTurn()
		g.Submit(p, harness.Order(defs.OrderType_UnitUngarrison, []uint64{u.InternalID}, 0, false), harness.Order(defs.OrderType_BuildingAttackUnit, []uint64{b.InternalID}, int64(target.InternalID), false))
		g.EndTurn()
		after := g.State(p).Unit(u.InternalID)
		if after.GarrisonedIn != nil {
			t.Fatal("fixture did not ungarrison")
		}
		harness.AssertTickSpend(t, after.TickActions, 1)
		skipped := false
		for _, a := range after.TickActions {
			if a.Order.Source == "building_garrison" && a.Tick == 1 && a.Execute.Outcome == "skipped" && a.Execute.Reason == "own_action_preempted_garrison_attack" && a.Execute.StaminaSpent == 0 {
				skipped = true
			}
		}
		if !skipped {
			t.Fatal("preempted garrison attack missing")
		}
	})
	t.Run("capacity-race", func(t *testing.T) {
		g := garrisonCombatGame(t, true)
		p := g.Human(0)
		ids, b := []uint64{}, g.Self(p).Buildings[0].InternalID
		for _, u := range g.Self(p).Units {
			ids = append(ids, u.InternalID)
			passive(g, p, u.InternalID)
		}
		passive(g, g.Human(1), g.Self(g.Human(1)).Units[0].InternalID)
		g.Submit(p, harness.OrderGarrison(ids, b))
		g.EndTurn()
		winners := 0
		for _, id := range ids {
			u := g.State(p).Unit(id)
			if u.GarrisonedIn == nil {
				a := harness.RequireTickAction(t, u.TickActions, "garrison")
				if a.Execute.Outcome != "blocked" || a.Execute.Reason != "garrison_full" {
					t.Errorf("losing entry outcome missing: %+v", a)
				}
				harness.AssertTickSpend(t, u.TickActions, 1)
			} else {
				winners++
				entry := harness.RequireTickAction(t, u.TickActions, "garrison")
				if entry.Execute.StaminaSpent != 1 {
					t.Errorf("entry spent %d stamina, want 1", entry.Execute.StaminaSpent)
				}
			}
		}
		if winners != 1 {
			t.Fatalf("fixture has %d garrison winners", winners)
		}
	})
	g := garrisonCombatGame(t)
	p, enemy := g.Human(0), g.Human(1)
	u, target := g.Self(p).Units[0], g.Self(enemy).Units[0]
	b := g.Self(p).Buildings[0]
	g.Submit(p, harness.OrderGarrison([]uint64{u.InternalID}, b.InternalID))
	g.EndTurn()
	if g.Self(p).Units[0].GarrisonedIn == nil {
		t.Fatal("fixture did not garrison")
	}
	before := g.Self(enemy).Units[0].CombatStats.Health
	g.Submit(p, harness.Order(defs.OrderType_BuildingAttackUnit, []uint64{b.InternalID}, int64(target.InternalID), false))
	g.EndTurn()
	if g.Self(enemy).Units[0].CombatStats.Health >= before {
		t.Fatal("fixture did not attack")
	}
	attacks := 0
	for _, a := range g.Self(p).Units[0].TickActions {
		if a.Order.Source != "building_garrison" {
			continue
		}
		attacks++
		if a.Tick < 1 || a.Intent.Kind != "attack" || a.Execute.Outcome != "executed" || a.Execute.StaminaSpent <= 0 || a.Execute.Target != (harness.TickTarget{Kind: "unit", InternalID: target.InternalID}) {
			t.Errorf("incorrect garrison attack: %+v", a)
		}
	}
	if attacks == 0 {
		t.Fatal("building-initiated attack missing from unit history")
	}
}

func garrisonDamageWithBonus(t *testing.T, attack defs.AttackType, bonus defs.TechConfig, fallback_attack ...int) float64 {
	t.Helper()
	g := garrisonCombatGameTweaked(t, func() {
		defs.BonusConfigs = []defs.TechConfig{bonus}
		unit := defs.UnitConfigs[22]
		unit.Attack_type = attack
		unit.Attack_blunt, unit.Attack_piercing = 0, 0
		switch attack {
		case defs.AttackType_BLUNT:
			unit.Attack_blunt = 20
		case defs.AttackType_PIERCING:
			unit.Attack_piercing = 20
		case defs.AttackType_MAGIC:
			defs.BonusConfigs = append(defs.BonusConfigs, defs.TechConfig{Affects_unit_ids: []uint32{22}, Bonus: defs.CombatBonus{Attack_magic: 20}})
		}
		unit.Defense_blunt, unit.Defense_piercing = 0, 0
		unit.Max_health = 10000
		defs.UnitConfigs[22] = unit
		building := defs.BuildingConfigs[207]
		building.Attack_type = defs.AttackType_BLUNT_PIERCING_MAGIC
		building.Attack_blunt, building.Attack_piercing = 0, 0
		cap := 2
		building.Max_garrison_attack_blunt, building.Max_garrison_attack_piercing, building.Max_garrison_attack_magic = &cap, &cap, &cap
		if len(fallback_attack) > 0 {
			building.Attack_blunt, building.Attack_piercing = fallback_attack[0], fallback_attack[0]
			building.Max_garrison_attack_blunt, building.Max_garrison_attack_piercing = nil, nil
		}
		defs.BuildingConfigs[207] = building
	})
	p, enemy := g.Human(0), g.Human(1)
	u, target, b := g.Self(p).Units[0], g.Self(enemy).Units[0], g.Self(p).Buildings[0]
	passive(g, p, u.InternalID)
	passive(g, enemy, target.InternalID)
	g.Submit(p, harness.OrderGarrison([]uint64{u.InternalID}, b.InternalID))
	g.EndTurn()
	if g.Self(p).Units[0].GarrisonedIn == nil {
		t.Fatal("cap bonus fixture did not garrison")
	}
	g.Submit(p, harness.Order(defs.OrderType_BuildingAttackUnit, []uint64{b.InternalID}, int64(target.InternalID), false))
	g.EndTurn()
	damage := 0.0
	for _, action := range g.Self(p).Units[0].TickActions {
		for _, effect := range action.Execute.Effects {
			if effect.Kind == "damage" && effect.Target.InternalID == target.InternalID {
				damage += effect.Amount
			}
		}
	}
	if damage <= 0 {
		t.Fatal("cap bonus fixture did not attack")
	}
	return damage
}

func TestGarrisonAttackCapBonusesByDamageType(t *testing.T) {
	cases := []struct {
		name   string
		attack defs.AttackType
		bonus  defs.TechConfig
	}{
		{"blunt", defs.AttackType_BLUNT, defs.TechConfig{Bonus_garrison_attack_cap_blunt: 2}},
		{"piercing", defs.AttackType_PIERCING, defs.TechConfig{Bonus_garrison_attack_cap_piercing: 2}},
		{"magic", defs.AttackType_MAGIC, defs.TechConfig{Bonus_garrison_attack_cap_magic: 2}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			baseline := garrisonDamageWithBonus(t, test.attack, defs.TechConfig{})
			for _, candidate := range cases {
				bonus := candidate.bonus
				bonus.Affects_building_ids = []uint32{207}
				damage := garrisonDamageWithBonus(t, test.attack, bonus)
				if candidate.attack == test.attack && damage <= baseline {
					t.Errorf("matching cap bonus did not increase damage: %v <= %v", damage, baseline)
				}
				if candidate.attack != test.attack && damage != baseline {
					t.Errorf("%s cap changed %s damage: %v != %v", candidate.name, test.name, damage, baseline)
				}
			}
		})
	}
}

func TestOmittedGarrisonCapsUseCurrentBuildingAttack(t *testing.T) {
	bonus := defs.TechConfig{
		Affects_building_ids:               []uint32{207},
		Bonus:                              defs.CombatBonus{Attack_blunt: 2, Attack_piercing: 2},
		Bonus_garrison_attack_cap_blunt:    2,
		Bonus_garrison_attack_cap_piercing: 2,
	}
	for _, attack := range []defs.AttackType{defs.AttackType_BLUNT, defs.AttackType_PIERCING} {
		damage := garrisonDamageWithBonus(t, attack, bonus, 5)
		if math.Abs(damage-7.2) > 1e-9 {
			t.Errorf("attack type %d: garrison damage = %v, want 7.2 from cap 9", attack, damage)
		}
	}
}

func TestGarrisonSwings(t *testing.T) {
	g := garrisonCombatGame(t)
	p0, p1 := g.Human(0), g.Human(1)
	u0 := g.Self(p0).Units[0]
	u1 := g.Self(p1).Units[0]
	b0 := g.Self(p0).Buildings[0]

	// Garrison u0 in b0
	g.Submit(p0, harness.OrderGarrison([]uint64{u0.InternalID}, b0.InternalID))
	g.Submit(p1) // p1 does nothing

	// Now u0 is garrisoned. Order building to attack u1.
	before := g.Self(p1).Units[0].CombatStats.Health
	g.Submit(p0, harness.Order(defs.OrderType_BuildingAttackUnit, []uint64{b0.InternalID}, int64(u1.InternalID), false))
	g.Submit(p1) // u1 does nothing

	damage := before - g.Self(p1).Units[0].CombatStats.Health
	// Building 21 has 0 attack, but max garrison piercing 3. u0 has 9 piercing.
	// So capped at 3 piercing. u1 (id 13) has 3 piercing defense. 3 - 3 = 0.
	// But damage is clamped to positive due to floor.
	if damage <= 0 || math.IsNaN(damage) {
		t.Errorf("expected positive damage from garrison attack, got %v", damage)
	}
}
