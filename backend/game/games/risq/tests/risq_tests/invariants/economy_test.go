package invariants

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func economyScenario() scenario {
	return scenario{players: 3, turns: 7, orders: economyOrders, verify: verifyEconomy,
		setup: func(t *testing.T) *harness.Game {
			return invariantGame(t, 3, homeSpaces(3, 3)+centralSpace, testRegions, noConfig)
		},
	}
}
