package combat

import (
	"fmt"
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

// A blunt infantry against an idle heavy infantry (50hp, so it never dies) at the placement
func stanceGame(t *testing.T, enemy enemyPlacement) *harness.Game {
	t.Helper()
	attacker := unitZoneJSON(attackerZone, 11, 0)
	defender := unitZoneJSON(enemy.zone, 13, 1)
	zones := attacker + "," + defender
	if enemy.zone == attackerZone && enemy.space == (harness.Coord{}) {
		zones = `{"x":1,"y":0,"units":[{"id":11,"player":0,"count":1},{"id":13,"player":1,"count":1}]}`
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
