package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
	"github.com/gin-gonic/gin"
)

func attackCleanupGame(t *testing.T) *harness.Game {
	t.Helper()
	doc := `{"board_size":2,"players":2,"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":23,"player":0},"units":[{"id":1,"player":0,"count":1},{"id":1,"player":1,"count":2}]},{"x":1,"y":0,"building":{"id":2,"player":1}}]},{"x":1,"y":0,"terrain":1},{"x":2,"y":0,"terrain":1}]}`
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"cleanup": doc})
	config := defs.BuildingConfigs[23]
	config.Attack_blunt, config.Attack_piercing, config.Attack_range = 0, 0, defs.RisqRange_SPACE
	config.Vision.Secondary = defs.VisibilityGood
	defs.BuildingConfigs[23] = config
	unit := defs.UnitConfigs[1]
	unit.Vision.Secondary, unit.Vision.Adjacent = defs.VisibilityGood, defs.VisibilityGood
	defs.UnitConfigs[1] = unit
	return harness.NewGame(t, "custom:cleanup", 1, 2)
}

func TestBuildingAttackDropsInvalidTargets(t *testing.T) {
	for _, outcome := range []string{"dead", "out-of-range"} {
		for _, queued := range []bool{false, true} {
			t.Run(outcome+map[bool]string{false: "/front", true: "/queued"}[queued], func(t *testing.T) {
				g := attackCleanupGame(t)
				p, enemy := g.Human(0), g.Human(1)
				building, unit := buildingID(g, p, 23), unitIDs(g, enemy, 1)[0]
				g.Action(p, "set-building-behavior", gin.H{"internal_ids": []uint64{building}, "auto_attack": false})
				orders := []defs.OrderFromFrontend{}
				if queued {
					orders = append(orders, harness.Order(defs.OrderType_BuildingAttackBuilding, []uint64{building}, int64(buildingID(g, enemy, 2)), false))
				}
				orders = append(orders, harness.Order(defs.OrderType_BuildingAttackUnit, []uint64{building}, int64(unit), false))
				g.Submit(p, orders...)
				g.EndTurn()
				if len(g.Self(p).ActiveOrders) != len(orders) {
					t.Fatalf("setup did not queue attacks: %+v", g.Self(p))
				}
				target_order := harness.Order(defs.OrderType_UnitDelete, []uint64{unit}, 0, false)
				if outcome == "out-of-range" {
					target_order = harness.Order(defs.OrderType_UnitMoveSpace, []uint64{unit}, harness.SpaceKey(2, 0), false)
				}
				g.Submit(enemy, target_order)
				g.EndTurn()
				for turn := 0; outcome == "out-of-range" && turn < 4 && g.State(enemy).Unit(unit).Space.X != 2; turn++ {
					g.EndTurn()
				}
				if outcome == "out-of-range" && g.State(enemy).Unit(unit).Space.X != 2 {
					t.Fatal("target did not leave range")
				}
				want := 0
				if queued {
					want = 1
				}
				if len(g.Self(p).ActiveOrders) != want || len(entity(g, p, "buildings", building)["active_orders"].([]gin.H)) != want {
					t.Fatalf("stale attack survived: %+v", g.Self(p).ActiveOrders)
				}
			})
		}
	}
}
