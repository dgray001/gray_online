package combat

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

const (
	blunt, heavy             = 11, 13
	flat, mountains          = 1, 101
	stationary, closingIn    = "stationary", "closing in"
	meetingBoard             = `{"board_size":1,"players":2,"spaces":[%s]}`
	unitJSON, zoneJSON       = `{"id":%d,"player":%d,"count":1}`, `{"x":%d,"y":%d,"units":[%s]}`
	spaceJSON                = `{"x":%d,"y":%d,"terrain":%d,"zones":[%s]}`
	attackerSlot, targetSlot = 0, 1
)

// A unit placed in a zone, for building meeting scenarios
type placed struct{ id, player, zx, zy int }

// One space holding the given units; units sharing a zone coordinate share the zone
func spaceOf(x, y, terrain int, units ...placed) string {
	var order [][2]int
	byZone := map[[2]int][]string{}
	for _, u := range units {
		key := [2]int{u.zx, u.zy}
		if byZone[key] == nil {
			order = append(order, key)
		}
		byZone[key] = append(byZone[key], fmt.Sprintf(unitJSON, u.id, u.player))
	}
	zones := make([]string, len(order))
	for i, key := range order {
		zones[i] = fmt.Sprintf(zoneJSON, key[0], key[1], strings.Join(byZone[key], ","))
	}
	return fmt.Sprintf(spaceJSON, x, y, terrain, strings.Join(zones, ","))
}

func meetingGame(t *testing.T, spaces ...string) *harness.Game {
	t.Helper()
	doc := fmt.Sprintf(meetingBoard, strings.Join(spaces, ","))
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"meeting": doc})
	return harness.NewGame(t, "custom:meeting", 1, 2)
}

// The unit of the player in the given slot, which holds one unit
func soleUnit(g *harness.Game, slot int) harness.Unit {
	return g.Self(g.Human(slot)).Units[0]
}

func attackOrder(attacker, target harness.Unit) defs.OrderFromFrontend {
	return harness.OrderAttackUnit([]uint64{attacker.InternalID}, target.InternalID)
}
