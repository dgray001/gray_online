package sims

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness/aisim"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

const emptyModel = `{"rules":[]}`
const remote = `{"x":-3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":1},"units":[{"id":1,"player":1,"count":1}]}]}`

func once(actions string) string {
	return `{"rules":[{"when":{"turn_equals":{"amount":1}},"then":[` + actions + `]}]}`
}

func fixture(t *testing.T, bank, spaces, extra string, models ...string) *aisim.Game {
	t.Helper()
	document := fmt.Sprintf(`{"board_size":4,"players":%d,"space_gold_income":0,"starting_bank":%s,"spaces":[%s]%s}`, len(models), bank, spaces, extra)
	fakeboard.UseConfig(t, "../../../config", nil, map[string]string{"ai-test": document})
	return aisim.New(t, "custom:ai-test", models...)
}

func TestEmptyModelsResolveTurnsAndStop(t *testing.T) {
	g := fixture(t, `{}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":2,"player":0},"units":[{"id":1,"player":0,"count":1}]}]},`+remote, "", emptyModel, emptyModel)
	g.Run(t, 3)
	if len(g.Trace) != 6 {
		t.Fatalf("actions=%+v", g.Trace)
	}
	g.Stop(t)
	g.Stop(t)
}
