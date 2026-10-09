package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

const home = `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":0},"units":[{"id":1,"player":0,"count":2},{"id":11,"player":0,"count":1}]},{"x":1,"y":0,"building":{"id":11,"player":0}},{"x":-1,"y":0,"building":{"id":3,"player":0}},{"x":0,"y":1,"building":{"id":22,"player":0}},{"x":0,"y":-1,"building":{"id":23,"player":0}},{"x":1,"y":-1,"resource":11}]}`
const route = `,{"x":1,"y":0,"terrain":1},{"x":2,"y":0,"terrain":1},{"x":3,"y":0,"terrain":1}`
const enemy = `,{"x":-3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":1},"units":[{"id":1,"player":1,"count":1}]}]}`

func useOrderConfig(t *testing.T, bank, extra string) {
	t.Helper()
	doc := `{"board_size":3,"players":2,"space_gold_income":0,"unlimited_population":true,"starting_bank":` + bank + `,"spaces":[` + home + route + enemy + `]` + extra + `}`
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"orders": doc})
}

func orderGame(t *testing.T, bank, extra string) *harness.Game {
	t.Helper()
	useOrderConfig(t, bank, extra)
	return harness.NewGame(t, "custom:orders", 1, 2)
}
