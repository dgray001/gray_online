package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func scenarioGame(t *testing.T, spaces string) *harness.Game {
	t.Helper()
	doc := `{"board_size":2,"players":2,"starting_bank":{"wood":120},"spaces":[` + spaces + `]}`
	fakeboard.UseConfig(t, shippedConfig, nil, map[string]string{"scenario": doc})
	return harness.NewGame(t, "custom:scenario", 1, 2)
}

func centerHousing(g *harness.Game, human int) harness.Building {
	g.T.Helper()
	for _, b := range g.Self(human).Buildings {
		if b.Space == (harness.Coord{}) && b.Zone == (harness.Coord{}) && b.BuildingID == 2 {
			return b
		}
	}
	g.T.Fatal("housing missing from center zone")
	return harness.Building{}
}
