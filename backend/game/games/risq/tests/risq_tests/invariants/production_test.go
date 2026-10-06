package invariants

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func productionScenario() scenario {
	return scenario{players: 2, turns: 4, orders: productionOrders, verify: verifyProduction,
		setup: func(t *testing.T) *harness.Game {
			return invariantGame(t, 2, pairedCenters(5), "", func() {
				config := defs.BuildingConfigs[1]
				config.Population_support = 3
				defs.BuildingConfigs[1] = config
			})
		},
	}
}
