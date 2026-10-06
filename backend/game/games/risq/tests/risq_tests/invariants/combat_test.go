package invariants

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

const fighters = `,{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":13,"player":0,"count":1},{"id":13,"player":1,"count":1}]}]}`

func combatScenario() scenario {
	return scenario{players: 2, turns: 4, orders: combatOrders, verify: verifyCombat,
		setup: func(t *testing.T) *harness.Game {
			return invariantGame(t, 2, homeSpaces(2, 1)+fighters, testRegions, func() {
				config := defs.UnitConfigs[13]
				config.Max_health, config.Turn_stamina = 8, 3
				config.Attack_type, config.Attack_blunt = defs.AttackType_BLUNT, 10
				config.Defense_blunt, config.Penetration_blunt = 0, 0
				defs.UnitConfigs[13] = config
			})
		},
	}
}
