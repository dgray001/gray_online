package invariants

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

const sharedWorkers = `,{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"resource":1,"units":[{"id":1,"player":0,"count":1},{"id":1,"player":1,"count":1}]}]}`

func gatheringScenario() scenario {
	return scenario{players: 2, turns: 3, orders: gatheringOrders, verify: verifyGathering,
		setup: func(t *testing.T) *harness.Game {
			return invariantGame(t, 2, homeSpaces(2, 1)+sharedWorkers, testRegions, func() {
				config := defs.ResourceConfigs[1]
				config.Starting_resources = 3.5
				defs.ResourceConfigs[1] = config
			})
		},
	}
}
