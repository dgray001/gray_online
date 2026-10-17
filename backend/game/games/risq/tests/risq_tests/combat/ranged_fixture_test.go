package combat

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
	"github.com/gin-gonic/gin"
)

func rangedGame(t *testing.T, distance int, nearbyVillager bool) *harness.Game {
	t.Helper()
	return rangedGameWithRange(t, distance, nearbyVillager, defs.RisqRange_ADJACENT)
}

func rangedGameWithRange(t *testing.T, distance int, nearbyVillager bool, attack_range defs.RisqRange) *harness.Game {
	t.Helper()
	var spaces []string
	for x := 0; x <= distance; x++ {
		zones := ""
		if x == 0 {
			units := `{"id":101,"player":0,"count":1}`
			if nearbyVillager {
				units += `,{"id":1,"player":1,"count":1}`
			}
			zones = `{"x":0,"y":0,"units":[` + units + `]}`
		} else if x == distance {
			zones = unitZoneJSON(harness.Coord{}, 103, 1)
		}
		spaces = append(spaces, fmt.Sprintf(`{"x":%d,"y":0,"terrain":1,"zones":[%s]}`, x, zones))
	}
	doc := fmt.Sprintf(`{"board_size":3,"players":2,"spaces":[%s]}`, strings.Join(spaces, ","))
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"ranged": doc})
	config := defs.UnitConfigs[101]
	config.Attack_range = attack_range
	config.Vision.Adjacent, config.Vision.Edge_opposite, config.Vision.Secondary = defs.VisibilityGood, defs.VisibilityGood, defs.VisibilityGood
	defs.UnitConfigs[101] = config
	g := harness.NewGame(t, "custom:ranged", 1, 2)
	for _, unit := range g.Self(g.Human(1)).Units {
		if unit.UnitID == 103 {
			g.Action(g.Human(1), "set-unit-behavior", gin.H{"internal_ids": []uint64{unit.InternalID}, "stance": uint8(defs.UnitStance_PASSIVE), "attack_back": false})
		}
	}
	return g
}

func TestRangedStancesHoldPosition(t *testing.T) {
	for _, c := range []struct {
		name        string
		stance      defs.UnitStance
		distance    int
		attacks     bool
		attackRange defs.RisqRange
	}{
		{"defensive-adjacent", defs.UnitStance_DEFENSIVE, 1, true, defs.RisqRange_ADJACENT},
		{"stand-ground-adjacent", defs.UnitStance_STAND_GROUND, 1, true, defs.RisqRange_ADJACENT},
		{"passive-adjacent", defs.UnitStance_PASSIVE, 1, false, defs.RisqRange_ADJACENT},
		{"defensive-out-of-range", defs.UnitStance_DEFENSIVE, 2, false, defs.RisqRange_ADJACENT},
		{"stand-ground-out-of-range", defs.UnitStance_STAND_GROUND, 2, false, defs.RisqRange_ADJACENT},
		{"aggressive-out-of-range", defs.UnitStance_AGGRESSIVE, 2, false, defs.RisqRange_ADJACENT},
		{"aggressive-range-two", defs.UnitStance_AGGRESSIVE, 2, true, defs.RisqRange_SECONDARY},
		{"defensive-range-two", defs.UnitStance_DEFENSIVE, 2, true, defs.RisqRange_SECONDARY},
		{"stand-ground-range-two", defs.UnitStance_STAND_GROUND, 2, true, defs.RisqRange_SECONDARY},
	} {
		t.Run(c.name, func(t *testing.T) {
			g := rangedGameWithRange(t, c.distance, false, c.attackRange)
			owner, enemy := g.Human(0), g.Human(1)
			attacker, target := g.Self(owner).Units[0], g.Self(enemy).Units[0]
			g.Action(owner, "set-unit-behavior", gin.H{"internal_ids": []uint64{attacker.InternalID}, "stance": uint8(c.stance), "attack_back": false})
			for turn := 0; turn < 2; turn++ {
				g.EndTurn()
				unit := g.Self(owner).Units[0]
				if unit.Space != attacker.Space || unit.Zone != attacker.Zone {
					t.Fatalf("stance moved: %+v", unit)
				}
				if attacked := g.Self(enemy).Units[0].CombatStats.Health < target.CombatStats.Health; attacked != c.attacks {
					t.Fatalf("attacked=%v, want %v", attacked, c.attacks)
				}
				if !c.attacks && len(unit.ActiveOrders) != 0 {
					t.Fatalf("out-of-range auto order: %+v", unit.ActiveOrders)
				}
			}
		})
	}
}

func TestRangedPriorityPrecedesDistance(t *testing.T) {
	for _, stance := range []defs.UnitStance{defs.UnitStance_DEFENSIVE, defs.UnitStance_STAND_GROUND} {
		t.Run(map[defs.UnitStance]string{defs.UnitStance_DEFENSIVE: "defensive", defs.UnitStance_STAND_GROUND: "stand-ground"}[stance], func(t *testing.T) {
			g := rangedGame(t, 1, true)
			owner, enemy := g.Human(0), g.Human(1)
			attacker := g.Self(owner).Units[0]
			g.Action(owner, "set-unit-behavior", gin.H{"internal_ids": []uint64{attacker.InternalID}, "stance": uint8(stance), "attack_back": false, "target_priority": []uint8{uint8(defs.TargetCategory_MILITARY), uint8(defs.TargetCategory_ECONOMIC)}})
			g.EndTurn()
			state := g.Self(enemy)
			if len(state.Units) != 2 {
				t.Fatal("nearby villager killed first")
			}
			for _, unit := range state.Units {
				if unit.UnitID == 103 && unit.CombatStats.Health >= float64(unit.CombatStats.MaxHealth) {
					t.Fatal("adjacent military target ignored")
				}
				if unit.UnitID == 1 && unit.CombatStats.Health != float64(unit.CombatStats.MaxHealth) {
					t.Fatal("nearby villager damaged first")
				}
			}
			unit := g.Self(owner).Units[0]
			if unit.Space != attacker.Space || unit.Zone != attacker.Zone {
				t.Fatalf("ranged unit moved: %+v", unit)
			}
		})
	}
}

func TestRangedDefensiveDoesNotFollowRetreatingTarget(t *testing.T) {
	g := rangedGame(t, 2, false)
	owner, enemy := g.Human(0), g.Human(1)
	attacker, target := g.Self(owner).Units[0], g.Self(enemy).Units[0]
	g.Action(owner, "set-unit-behavior", gin.H{"internal_ids": []uint64{attacker.InternalID}, "stance": uint8(defs.UnitStance_DEFENSIVE), "attack_back": false})
	g.Submit(enemy, harness.OrderMoveZone([]uint64{target.InternalID}, 1, 0, 0, 0))
	g.Submit(owner)
	if g.Self(enemy).Units[0].CombatStats.Health >= target.CombatStats.Health {
		t.Fatal("did not fire when enemy entered range")
	}
	g.Submit(enemy, harness.OrderMoveZone([]uint64{target.InternalID}, 2, 0, 0, 0))
	g.Submit(owner)
	if g.Self(enemy).Units[0].Space != (harness.Coord{X: 2}) {
		t.Fatal("enemy did not retreat beyond range")
	}
	unit := g.Self(owner).Units[0]
	if unit.Space != attacker.Space || unit.Zone != attacker.Zone || len(unit.ActiveOrders) != 0 {
		t.Fatalf("followed retreating target: %+v", unit)
	}
}
