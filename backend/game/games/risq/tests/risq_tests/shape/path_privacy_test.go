package shape

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestEnemyMovePathRequiresSpyVision(t *testing.T) {
	for _, level := range []uint8{3, 4} {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			g := configuredGame(t, level, func() {
				unit := defs.UnitConfigs[1]
				unit.Turn_stamina = 10
				defs.UnitConfigs[1] = unit
				scout := defs.UnitConfigs[15]
				scout.Vision.Secondary = level
				defs.UnitConfigs[15] = scout
			})
			id := unitID(g, 1, 0, 1)
			g.Submit(g.Human(1), harness.OrderMove([]uint64{id}, 3, 0))
			g.EndTurn()
			owner := player(t, snapshot(g, g.Human(1), false), 1)
			path := array(t, entity(t, owner, "units", id)["move_path"])
			if len(path) == 0 {
				t.Fatal("owner has no path to test privacy")
			}
			enemy := player(t, snapshot(g, g.Human(0), false), 1)
			u := entity(t, enemy, "units", id)
			if level == 4 {
				equal(t, u["move_path"], path)
			} else if _, exists := u["move_path"]; exists {
				t.Error("good visibility exposed enemy move path")
			}
		})
	}
}
