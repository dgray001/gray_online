package invariants

import (
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func constructionScenario() scenario {
	return scenario{players: 2, turns: 3, verify: verifyConstruction,
		setup: func(t *testing.T) *harness.Game {
			return invariantGame(t, 2, homeSpaces(2, 1)+strings.ReplaceAll(sharedWorkers, `"resource":1,`, ""), testRegions, noConfig)
		},
		orders: func(g *harness.Game, turn, slot int) []defs.OrderFromFrontend {
			if turn != 0 {
				return nil
			}
			ids := unitIDs(g, slot, 1)
			return []defs.OrderFromFrontend{harness.OrderBuild(ids[1:], 2, 0, 0, 0, 0)}
		},
	}
}
