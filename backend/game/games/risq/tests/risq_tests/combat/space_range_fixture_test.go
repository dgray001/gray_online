package combat

import (
	"fmt"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
	"strings"
	"testing"
)

func spaceRangeGame(t *testing.T, attackRange defs.RisqRange, spaces ...string) *harness.Game {
	t.Helper()
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"space-range": fmt.Sprintf(meetingBoard, strings.Join(spaces, ","))})
	for _, id := range []uint32{blunt, heavy} {
		config := defs.UnitConfigs[id]
		config.Attack_range = attackRange
		defs.UnitConfigs[id] = config
	}
	return harness.NewGame(t, "custom:space-range", 1, 2)
}

func spaceRangeAttackOrder(attacker, target harness.Unit, kind defs.OrderType) defs.OrderFromFrontend {
	targetID := int64(target.InternalID)
	if kind == defs.OrderType_UnitAttackZone {
		targetID = harness.ZoneKey(target.Space.X, target.Space.Y, target.Zone.X, target.Zone.Y)
	} else if kind == defs.OrderType_UnitAttackSpace {
		targetID = harness.SpaceKey(target.Space.X, target.Space.Y)
	}
	return harness.Order(kind, []uint64{attacker.InternalID}, targetID, false)
}
