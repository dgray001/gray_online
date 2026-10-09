package economy

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func productionGame(t *testing.T, workers, centers int, extra string) *harness.Game {
	t.Helper()
	spaces := []string{fmt.Sprintf(`{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":0},"units":[{"id":1,"player":0,"count":%d}]}]}`, workers)}
	for x := 1; x < centers; x++ {
		spaces = append(spaces, fmt.Sprintf(`{"x":%d,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":0}}]}`, x))
	}
	spaces = append(spaces, `{"x":-2,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":1,"count":1}]}]}`)
	doc := `{"board_size":3,"players":2,"starting_bank":{"food":300,"wood":300,"stone":100,"gold":500}` + extra + `,"spaces":[` + strings.Join(spaces, ",") + `]}`
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"production": doc})
	return harness.NewGame(t, "custom:production", 1, 2)
}

func centers(g *harness.Game, human int) []harness.Building {
	buildings := g.Self(human).Buildings
	if len(buildings) == 2 && buildings[0].Space.X > buildings[1].Space.X {
		buildings[0], buildings[1] = buildings[1], buildings[0]
	}
	return buildings
}
