package combat

import (
	"fmt"
	"math"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func TestHealthNetting(t *testing.T) {
	g := combatGame(t, 22, 22)
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
	if u := risq.ComputeUnitVsUnitTTK(21, 21); !u.Possible || u.Turns <= 0 || u.Ticks <= 0 {
		t.Fatalf("unexpected unit TTK: %+v", u)
	}
	if b := risq.ComputeUnitVsBuildingTTK(21, 202); !b.Possible || b.Ticks <= 0 {
		t.Fatalf("unexpected building TTK: %+v", b)
	}
	if ut := risq.ComputeUnitVsUnitTTKWithTechs(21, 21, []uint32{302}, []uint32{303}); !ut.Possible {
		t.Fatalf("unexpected tech TTK: %+v", ut)
	}
	if bt := risq.ComputeUnitVsBuildingTTKWithTechs(21, 202, []uint32{302}); !bt.Possible {
		t.Fatalf("unexpected building tech TTK: %+v", bt)
	}
	if len(risq.AllUnitIDs()) == 0 || len(risq.AllBuildingIDs()) == 0 {
		t.Fatal("empty entity ID lists")
	}
	if risq.UnitName(11) != "Test Worker" || risq.BuildingName(201) != "Test Town Hall" {
		t.Fatalf("unexpected display names: %q, %q", risq.UnitName(11), risq.BuildingName(201))
	}
}

func TestAttackTypesAndPenetration(t *testing.T) {
	g := combatGame(t, 21, 12)
	p0, p1 := g.Human(0), g.Human(1)
	u0 := g.Self(p0).Units[0]
	u1 := g.Self(p1).Units[0]
	g.Submit(p0, harness.OrderAttackUnit([]uint64{u0.InternalID}, u1.InternalID))
	g.Submit(p1) // do nothing
	after1 := g.Self(p1).Units[0]
	damage := u1.CombatStats.Health - after1.CombatStats.Health
	// ratio 2/8 is under the knee: (8-2) * stamina / 10
	if want := 6 * float64(u0.CurrentStamina) / 10; math.Abs(damage-want) > 1e-6 {
		t.Errorf("expected damage %v, got %v", want, damage)
	}
}

func TestPenetrationReducesDefense(t *testing.T) {
	g := combatGame(t, 42, 22)
	p0, p1 := g.Human(0), g.Human(1)
	u0 := g.Self(p0).Units[0]
	u1 := g.Self(p1).Units[0]
	g.Submit(p0, harness.OrderAttackUnit([]uint64{u0.InternalID}, u1.InternalID))
	g.Submit(p1)
	damage := u1.CombatStats.Health - g.Self(p1).Units[0].CombatStats.Health
	defense := 6 * (1 - 0.08)
	want := (16 - defense) * float64(u0.CurrentStamina) / 10
	if math.Abs(damage-want) > 1e-6 {
		t.Errorf("expected damage %v, got %v", want, damage)
	}
}

func splashGame(t *testing.T, enemies, friends int) *harness.Game {
	t.Helper()
	doc := fmt.Sprintf(`{"board_size":1,"players":2,"starting_techs":[306],"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":21,"player":0,"count":%d},{"id":21,"player":1,"count":%d}]}]}]}`, 1+friends, enemies)
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"combat": doc})
	return harness.NewGame(t, "custom:combat", 1, 2)
}

func TestSplashTargetSelection(t *testing.T) {
	cases := []struct {
		name                           string
		enemies, friends               int
		wantEnemiesHit, wantFriendsHit int
	}{
		{"enemies-before-friendlies", 3, 1, 2, 0},
		{"friendly-fire-when-no-enemies-remain", 1, 2, 0, 2},
		{"capped-by-max-targets", 5, 0, 2, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := splashGame(t, c.enemies, c.friends)
			p, enemy := g.Human(0), g.Human(1)
			attacker, friends, enemies := g.Self(p).Units[0], g.Self(p).Units[1:], g.Self(enemy).Units
			for _, u := range enemies {
				passive(g, enemy, u.InternalID)
			}
			for _, u := range friends {
				passive(g, p, u.InternalID)
			}
			g.Submit(p, harness.OrderAttackUnit([]uint64{attacker.InternalID}, enemies[0].InternalID))
			g.EndTurn()
			states := map[int]harness.State{p: g.State(p), enemy: g.State(enemy)}
			lost := func(human int, u harness.Unit) float64 { return recordedDamage(states[human], u) }
			main_loss := lost(enemy, enemies[0])
			enemies_hit, friends_hit := 0, 0
			for _, u := range enemies[1:] {
				if loss := lost(enemy, u); loss > 0 {
					enemies_hit++
					if math.Abs(loss-0.5*main_loss) > 1e-3 {
						t.Errorf("splash loss %v, want half of %v", loss, main_loss)
					}
				}
			}
			for _, u := range friends {
				if lost(p, u) > 0 {
					friends_hit++
				}
			}
			if enemies_hit != c.wantEnemiesHit || friends_hit != c.wantFriendsHit {
				t.Errorf("hit %d enemies and %d friendlies, want %d and %d", enemies_hit, friends_hit, c.wantEnemiesHit, c.wantFriendsHit)
			}
		})
	}
}

func recordedDamage(state harness.State, u harness.Unit) float64 {
	total := 0.0
	for _, action := range state.Unit(u.InternalID).TickActions {
		for _, effect := range action.Execute.Effects {
			if effect.Kind == "damage" && effect.Target == (harness.TickTarget{Kind: "unit", InternalID: u.InternalID}) {
				total += effect.Amount
			}
		}
	}
	return total
}

func TestSplashTargetsRotateByTurn(t *testing.T) {
	g := splashGame(t, 5, 0)
	p, enemy := g.Human(0), g.Human(1)
	attacker, enemies := g.Self(p).Units[0], g.Self(enemy).Units
	for _, u := range enemies {
		passive(g, enemy, u.InternalID)
	}
	g.Submit(p, harness.OrderAttackUnit([]uint64{attacker.InternalID}, enemies[0].InternalID))
	candidates := enemies[1:]
	hitIndices := func() []int {
		hit := []int{}
		state := g.State(enemy)
		for i, u := range candidates {
			if recordedDamage(state, u) > 0 {
				hit = append(hit, i)
			}
		}
		return hit
	}
	g.EndTurn()
	first := hitIndices()
	g.EndTurn()
	second := hitIndices()
	if len(first) != 2 || len(second) != 2 {
		t.Fatalf("want 2 splash targets each turn, got %v then %v", first, second)
	}
	for i, index := range first {
		if want := (index + 1) % len(candidates); second[i] != want && second[(i+1)%2] != want {
			t.Errorf("turn 2 hit %v, want the window shifted one from %v", second, first)
		}
	}
}
