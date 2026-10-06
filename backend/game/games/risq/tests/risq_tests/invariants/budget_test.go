package invariants

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func budgetScenario() scenario {
	return scenario{players: 2, turns: 3, orders: productionOrders, verify: verifyBudget,
		setup: func(t *testing.T) *harness.Game {
			return invariantGame(t, 2, pairedCenters(1), "", func() {
				config := defs.UnitConfigs[1]
				config.Cost = defs.RisqResourceCost{Food: 500, Wood: 500, Stone: 500, Gold: 500}
				defs.UnitConfigs[1] = config
			})
		},
	}
}
