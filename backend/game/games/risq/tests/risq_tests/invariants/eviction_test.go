package invariants

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func evictionScenario() scenario {
	return scenario{players: 2, turns: 5, capLossTurn: 4, orders: evictionOrders, verify: verifyEviction,
		setup: func(t *testing.T) *harness.Game {
			return invariantGame(t, 2, homeSpaces(2, 3)+centralSpace, testRegions, func() {
				config := defs.BuildingConfigs[1]
				config.Garrison_capacity = 1
				defs.BuildingConfigs[1] = config
			})
		},
	}
}
