package invariants

import (
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func farmScenario() scenario {
	return scenario{players: 2, turns: 4, orders: farmOrders, verify: verifyFarm,
		setup: func(t *testing.T) *harness.Game {
			spaces := strings.Replace(homeSpaces(2, 1), `"resource":11`, `"building":{"id":3,"player":0}`, 1)
			spaces = strings.Replace(spaces, `"resource":11`, `"building":{"id":3,"player":1}`, 1)
			return invariantGame(t, 2, spaces+centralSpace, "", func() {
				config := defs.BuildingConfigs[3]
				config.Gather.Starting_resources = 2.5
				defs.BuildingConfigs[3] = config
			})
		},
	}
}
