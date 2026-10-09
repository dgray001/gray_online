package economy

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

const remoteVillager = `{"x":-2,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":1,"count":1}]}]}`

func economyGame(t *testing.T, bank, spaces, rules string) *harness.Game {
	t.Helper()
	doc := `{"board_size":3,"players":2,"space_gold_income":0,"starting_bank":` + bank + `,"spaces":[` + spaces + `]` + rules + `}`
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"economy": doc})
	return harness.NewGame(t, "custom:economy", 1, 2)
}

func farmGame(t *testing.T, wood, workers int) *harness.Game {
	t.Helper()
	bank := fmt.Sprintf(`{"wood":%d}`, wood)
	space := fmt.Sprintf(`{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":3,"player":0},"units":[{"id":1,"player":0,"count":%d}]}]}`, workers)
	return economyGame(t, bank, space+","+remoteVillager, "")
}
