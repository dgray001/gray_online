package economy

import (
	"fmt"
	"os"
	"runtime/debug"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	testconfig "github.com/dgray001/gray_online/game/games/risq/tests/config"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

const remoteVillager = `{"x":-2,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":1,"count":1}]}]}`

func economyGame(t *testing.T, bank, spaces, rules string) *harness.Game {
	t.Helper()
	doc := `{"board_size":3,"players":2,"space_gold_income":0,"starting_bank":` + bank + `,"spaces":[` + spaces + `]` + rules + `}`
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"economy": doc})
	return startGame(t, "custom:economy", 1, 2)
}

func farmGame(t *testing.T, wood, workers int) *harness.Game {
	t.Helper()
	bank := fmt.Sprintf(`{"wood":%d}`, wood)
	space := fmt.Sprintf(`{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":3,"player":0},"units":[{"id":1,"player":0,"count":%d}]}]}`, workers)
	return economyGame(t, bank, space+","+remoteVillager, "")
}

// Economy games read plain state, so they skip per-tick histories and reuse reads within a turn
func startGame(t *testing.T, mapName string, seed int64, humans int) *harness.Game {
	t.Helper()
	g := harness.NewGame(t, mapName, seed, humans)
	g.Lite = true
	return g
}

func scenarioGame(t *testing.T, spaces string) *harness.Game {
	t.Helper()
	doc := `{"board_size":2,"players":2,"starting_bank":{"wood":120},"spaces":[` + spaces + `]}`
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"scenario": doc})
	return startGame(t, "custom:scenario", 1, 2)
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

var testConfig = testconfig.Dir()

func TestMain(m *testing.M) {
	debug.SetGCPercent(800)
	if err := defs.LoadConfig(testConfig); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
