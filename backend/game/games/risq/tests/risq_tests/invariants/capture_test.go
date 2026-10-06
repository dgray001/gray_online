package invariants

import (
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func captureScenario() scenario {
	return scenario{players: 2, turns: 4, capLossTurn: 3, orders: captureOrders, verify: verifyCapture,
		setup: func(t *testing.T) *harness.Game {
			spaces := strings.Replace(homeSpaces(2, 1), `"units":[{"id":1,`, `"units":[{"id":11,`, 1)
			g := invariantGame(t, 2, spaces+centralSpace, testRegions, noConfig)
			g.Action(g.Human(0), "set-unit-behavior", gin.H{"internal_ids": unitIDs(g, 0, 11), "stance": 1, "attack_back": false})
			return g
		},
	}
}
