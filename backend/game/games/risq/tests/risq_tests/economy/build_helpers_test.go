package economy

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func buildGame(t *testing.T, workers int) *harness.Game {
	t.Helper()
	doc := fmt.Sprintf(`{"board_size":3,"players":2,"starting_bank":{"wood":120},"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":0,"count":%d},{"id":11,"player":0,"count":1}]}]},{"x":1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":0,"count":1}]}]},{"x":2,"y":0,"terrain":1},{"x":-3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":1,"count":1}]}]}]}`, workers)
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"build": doc})
	return harness.NewGame(t, "custom:build", 1, 2)
}

func housingOrders(g *harness.Game, human int, x int) []defs.OrderFromFrontend {
	var orders []defs.OrderFromFrontend
	for _, u := range g.Self(human).Units {
		if u.UnitID == 1 {
			orders = append(orders, harness.OrderBuild([]uint64{u.InternalID}, 2, x, 0, 0, 0))
		}
	}
	return orders
}
