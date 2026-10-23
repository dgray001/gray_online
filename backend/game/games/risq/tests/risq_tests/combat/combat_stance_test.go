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

type enemyPlacement struct{ space, zone harness.Coord }

// The attacker stands in the edge zone facing (1,0), the only spot giving good vision of that neighbor
var (
	attackerZone  = harness.Coord{X: 1, Y: 0}
	sameZone      = enemyPlacement{harness.Coord{}, attackerZone}
	sameSpace     = enemyPlacement{harness.Coord{}, harness.Coord{X: -1, Y: 0}}
	adjacentSpace = enemyPlacement{harness.Coord{X: 1, Y: 0}, harness.Coord{}}
)

func unitZoneJSON(zone harness.Coord, unit_id int, player int) string {
	return fmt.Sprintf(`{"x":%d,"y":%d,"units":[{"id":%d,"player":%d,"count":1}]}`, zone.X, zone.Y, unit_id, player)
}

// A blunt grunt against an idle phalanx (50hp, so it never dies) at the placement
func stanceGame(t *testing.T, enemy enemyPlacement) *harness.Game {
	t.Helper()
	attacker := unitZoneJSON(attackerZone, 21, 0)
	defender := unitZoneJSON(enemy.zone, 22, 1)
	zones := attacker + "," + defender
	if enemy.zone == attackerZone && enemy.space == (harness.Coord{}) {
		zones = `{"x":1,"y":0,"units":[{"id":21,"player":0,"count":1},{"id":22,"player":1,"count":1}]}`
	}
	spaces := fmt.Sprintf(`{"x":0,"y":0,"terrain":1,"zones":[%s]}`, zones)
	if enemy.space != (harness.Coord{}) {
		spaces = fmt.Sprintf(`{"x":0,"y":0,"terrain":1,"zones":[%s]},{"x":%d,"y":%d,"terrain":1,"zones":[%s]}`, attacker, enemy.space.X, enemy.space.Y, defender)
	}
	doc := fmt.Sprintf(`{"board_size":1,"players":2,"spaces":[%s]}`, spaces)
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"stance": doc})
	return harness.NewGame(t, "custom:stance", 1, 2)
}

func TestStanceEngagement(t *testing.T) {
	cases := []struct {
		name    string
		stance  defs.UnitStance
		enemy   enemyPlacement
		engages bool
	}{
		{"passive ignores same zone", defs.UnitStance_PASSIVE, sameZone, false},
		{"stand ground hits same zone", defs.UnitStance_STAND_GROUND, sameZone, true},
		{"stand ground ignores same space", defs.UnitStance_STAND_GROUND, sameSpace, false},
		{"defensive hits same space", defs.UnitStance_DEFENSIVE, sameSpace, true},
		{"defensive ignores adjacent space", defs.UnitStance_DEFENSIVE, adjacentSpace, false},
		{"aggressive hits adjacent space", defs.UnitStance_AGGRESSIVE, adjacentSpace, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := stanceGame(t, c.enemy)
			p0, p1 := g.Human(0), g.Human(1)
			u0, u1 := g.Self(p0).Units[0], g.Self(p1).Units[0]
			g.Action(p0, "set-unit-behavior", gin.H{"internal_ids": []uint64{u0.InternalID}, "stance": uint8(c.stance)})
			g.Action(p1, "set-unit-behavior", gin.H{"internal_ids": []uint64{u1.InternalID}, "stance": uint8(defs.UnitStance_PASSIVE), "attack_back": false})
			g.Submit(p0)
			g.Submit(p1)
			engaged := g.Self(p1).Units[0].CombatStats.Health < u1.CombatStats.Health
			if engaged != c.engages {
				t.Errorf("engaged=%v, want %v", engaged, c.engages)
			}
			if c.engages {
				orders := g.Self(p0).Units[0].ActiveOrders
				if len(orders) != 1 || defs.OrderType(orders[0].OrderType) != defs.OrderType_UnitAutoAttackUnit {
					t.Fatalf("stance must create an auto-attack-unit order, got %+v", orders)
				}
			}
		})
	}
}

// Passive workers in the grunt's zone, elsewhere in its space and along a line of spaces, then a phalanx too tough to finish
func TestStanceRoamLimits(t *testing.T) {
	line := func(x int) harness.Coord { return harness.Coord{X: x} }
	workers := []enemyPlacement{{center, center}, {center, west}, {line(1), center}, {line(2), center}}
	for _, c := range []struct {
		name   string
		stance defs.UnitStance
		killed int
	}{{"stand ground", defs.UnitStance_STAND_GROUND, 1}, {"defensive", defs.UnitStance_DEFENSIVE, 2}, {"aggressive", defs.UnitStance_AGGRESSIVE, 4}} {
		t.Run(c.name, func(t *testing.T) {
			spaces := strings.Join([]string{
				spaceOf(0, 0, flat, placed{blunt, 0, 0, 0}, placed{villager, 1, 0, 0}, placed{villager, 1, -1, 0}),
				spaceOf(1, 0, flat, placed{villager, 1, 0, 0}),
				spaceOf(2, 0, flat, placed{villager, 1, 0, 0}),
				spaceOf(3, 0, flat, placed{heavy, 1, 0, 0}),
			}, ",")
			fakeboard.UseConfig(t, testConfig, nil, map[string]string{"roam": fmt.Sprintf(`{"board_size":3,"players":2,"spaces":[%s]}`, spaces)})
			fullVision()
			g := harness.NewGame(t, "custom:roam", 1, 2)
			owner, foe := g.Human(0), g.Human(1)
			passive(g, foe, unitWithID(g, foe, heavy).InternalID)
			g.Action(owner, "set-unit-behavior", gin.H{"internal_ids": []uint64{soleUnit(g, 0).InternalID}, "stance": uint8(c.stance), "attack_back": false})
			for turn := 0; turn < 10; turn++ {
				g.EndTurn()
				at := soleUnit(g, 0)
				if (c.stance == defs.UnitStance_STAND_GROUND && (at.Space != center || at.Zone != center)) || (c.stance == defs.UnitStance_DEFENSIVE && at.Space != center) {
					t.Fatalf("turn %d: %v left its allowed area, now at %v/%v", turn, c.name, at.Space, at.Zone)
				}
			}
			alive := map[enemyPlacement]bool{}
			for _, unit := range g.Self(foe).Units {
				alive[enemyPlacement{unit.Space, unit.Zone}] = true
			}
			for i, worker := range workers {
				if alive[worker] == (i < c.killed) {
					t.Errorf("worker at %v: alive=%v, want %v", worker, alive[worker], i >= c.killed)
				}
			}
			if end := soleUnit(g, 0); c.stance == defs.UnitStance_AGGRESSIVE && end.Space != line(3) {
				t.Errorf("aggressive unit stopped in %v instead of walking to the last space", end.Space)
			}
		})
	}
}

// A closer enemy across water is out of reach, so the aggressive grunt hits the reachable one
func TestAggressiveSkipsUnreachableTarget(t *testing.T) {
	spaces := strings.Join([]string{
		spaceOf(0, 0, flat, placed{blunt, 0, 0, 0}),
		spaceOf(1, 0, 251, placed{heavy, 1, 0, 0}),
		spaceOf(-1, 0, flat, placed{heavy, 1, 0, 0}),
	}, ",")
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"blocked": fmt.Sprintf(`{"board_size":2,"players":2,"spaces":[%s]}`, spaces)})
	fullVision()
	g := harness.NewGame(t, "custom:blocked", 1, 2)
	for _, foe := range g.Self(g.Human(1)).Units {
		passive(g, g.Human(1), foe.InternalID)
	}
	g.Action(g.Human(0), "set-unit-behavior", gin.H{"internal_ids": []uint64{soleUnit(g, 0).InternalID}, "stance": uint8(defs.UnitStance_AGGRESSIVE), "attack_back": false})
	for turn := 0; turn < 3; turn++ {
		g.EndTurn()
	}
	for _, foe := range g.Self(g.Human(1)).Units {
		if hurt(foe) != (foe.Space.X == -1) {
			t.Errorf("enemy at %v: hurt=%v", foe.Space, hurt(foe))
		}
	}
}

func TestStanceAttackBack(t *testing.T) {
	g := combatGame(t, 22, 22)
	p0, p1 := g.Human(0), g.Human(1)
	u0 := g.Self(p0).Units[0]
	u1 := g.Self(p1).Units[0]

	// Set u1 to PASSIVE and attack_back to true
	stance := uint8(defs.UnitStance_PASSIVE)
	attackBack := true
	g.Action(p1, "set-unit-behavior", gin.H{"internal_ids": []uint64{u1.InternalID}, "stance": stance, "attack_back": attackBack})

	g.Submit(p0, harness.OrderAttackUnit([]uint64{u0.InternalID}, u1.InternalID))
	g.Submit(p1) // Idle, should attack back

	if g.Self(p0).Units[0].CombatStats.Health == u0.CombatStats.Health {
		t.Errorf("u1 did not attack back")
	}
}

func TestPassiveNoAttackBack(t *testing.T) {
	g := combatGame(t, 22, 22)
	p0, p1 := g.Human(0), g.Human(1)
	u0, u1 := g.Self(p0).Units[0], g.Self(p1).Units[0]
	stance := uint8(defs.UnitStance_PASSIVE)
	attackBack := false
	g.Action(p1, "set-unit-behavior", gin.H{"internal_ids": []uint64{u1.InternalID}, "stance": stance, "attack_back": attackBack})
	g.Submit(p0, harness.OrderAttackUnit([]uint64{u0.InternalID}, u1.InternalID))
	g.Submit(p1)
	if g.Self(p0).Units[0].CombatStats.Health != u0.CombatStats.Health {
		t.Errorf("u1 attacked back despite attack_back=false")
	}
}

func attackBackTieGame(t *testing.T) *harness.Game {
	t.Helper()
	doc := `{"board_size":1,"players":2,"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":22,"player":0,"count":2},{"id":22,"player":1,"count":1}]}]}]}`
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"combat_tie": doc})
	return harness.NewGame(t, "custom:combat_tie", 1, 2)
}

func TestAttackBackRetainsTargetOnRangeTie(t *testing.T) {
	g := attackBackTieGame(t)
	p0, p1 := g.Human(0), g.Human(1)
	attackers, defender := g.Self(p0).Units, g.Self(p1).Units[0]
	g.Action(p1, "set-unit-behavior", gin.H{"internal_ids": []uint64{defender.InternalID}, "stance": uint8(defs.UnitStance_PASSIVE), "attack_back": true, "interrupt_current": true})
	g.Submit(p0, harness.OrderAttackUnit([]uint64{attackers[0].InternalID, attackers[1].InternalID}, defender.InternalID))
	g.Submit(p1)
	orders := g.Self(p1).ActiveOrders
	if len(orders) == 0 {
		t.Fatalf("defender has no active orders")
	}
	target1 := orders[0].TargetID
	g.Submit(p0, harness.OrderAttackUnit([]uint64{attackers[0].InternalID, attackers[1].InternalID}, defender.InternalID))
	g.Submit(p1)
	orders = g.Self(p1).ActiveOrders
	if len(orders) == 0 || orders[0].TargetID != target1 {
		t.Fatalf("defender retargeted on range tie: got %v want %v", orders[0].TargetID, target1)
	}
}
