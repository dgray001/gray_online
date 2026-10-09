package combat

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func combatGame(t *testing.T, unit1, unit2 uint32) *harness.Game {
	t.Helper()
	u1 := ""
	if unit1 != 0 {
		u1 = fmt.Sprintf(`{"id":%d,"player":0,"count":1}`, unit1)
	}
	u2 := ""
	if unit2 != 0 {
		u2 = fmt.Sprintf(`{"id":%d,"player":1,"count":1}`, unit2)
	}
	units := ""
	if u1 != "" && u2 != "" {
		units = u1 + "," + u2
	} else {
		units = u1 + u2
	}
	doc := fmt.Sprintf(`{"board_size":1,"players":2,"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[%s]}]}]}`, units)
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"combat": doc})
	return harness.NewGame(t, "custom:combat", 1, 2)
}

func garrisonCombatGame(t *testing.T) *harness.Game {
	t.Helper()
	// p0 gets an outpost (id 21) and a heavy infantry (id 13). p1 gets a unit 13.
	doc := `{"board_size":1,"players":2,"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":21,"player":0},"units":[{"id":13,"player":0,"count":1},{"id":13,"player":1,"count":1}]}]}]}`
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"combat": doc})
	return harness.NewGame(t, "custom:combat", 1, 2)
}

func combatGameTwoAttackers(t *testing.T) *harness.Game {
	t.Helper()
	doc := `{"board_size":1,"players":2,"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":0,"count":2},{"id":13,"player":1,"count":1}]}]}]}`
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"combat": doc})
	return harness.NewGame(t, "custom:combat", 1, 2)
}
