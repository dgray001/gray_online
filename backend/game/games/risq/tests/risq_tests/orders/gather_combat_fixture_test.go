package orders

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
	"github.com/gin-gonic/gin"
)

func gatherCombatGame(t *testing.T, producer uint32, owner int, enemyKind uint32) (*harness.Game, int) {
	t.Helper()
	doc := fmt.Sprintf(`{"board_size":1,"players":2,"unlimited_population":true,"space_gold_income":0,"starting_bank":%s,"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":%d,"player":0},"units":[{"id":1,"player":0,"count":1}]},{"x":1,"y":0,"building":{"id":2,"player":%d},"units":[{"id":%d,"player":1,"count":1}]}]}]}`, richBank, producer, owner, enemyKind)
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"point-combat": doc})
	g := harness.NewGame(t, "custom:point-combat", 1, 2)
	p := g.Human(0)
	g.Action(g.Human(1), "set-unit-behavior", gin.H{"internal_ids": unitIDs(g, g.Human(1), enemyKind), "stance": 1, "attack_back": false})
	return g, p
}
