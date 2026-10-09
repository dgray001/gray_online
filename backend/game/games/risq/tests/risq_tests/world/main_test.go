package world

import (
	"fmt"
	"os"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/config"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

var testConfig = testconfig.Dir()

func TestMain(m *testing.M) {
	if err := defs.LoadConfig(testConfig); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// A radius-4 two-slot custom map document; extra is more top-level fields, spaces the space entries
func mapDoc(extra string, spaces string) string {
	return `{"board_size":4,"players":2` + extra + `,"spaces":[` + spaces + `]}`
}

// A space entry with the given unit entries on its center zone
func spaceWith(x, y int, terrain int, units string) string {
	zones := ""
	if units != "" {
		zones = `,"zones":[{"x":0,"y":0,"units":[` + units + `]}]`
	}
	return fmt.Sprintf(`{"x":%d,"y":%d,"terrain":%d%s}`, x, y, terrain, zones)
}

func unit(id uint32, slot int) string {
	return fmt.Sprintf(`{"id":%d,"player":%d,"count":1}`, id, slot)
}

const (
	villager = 1
	infantry = 11
	grass    = 1
	hills    = 51
	water    = 152
)

// Starts a two-human game on the given map document, which exists only for this test
func startGame(t *testing.T, doc string) *harness.Game {
	t.Helper()
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"world": doc})
	return harness.NewGame(t, "custom:world", 1, 2)
}
