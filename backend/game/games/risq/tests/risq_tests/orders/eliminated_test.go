package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func TestEliminatedPlayersCannotSubmitAndDoNotBlockTurns(t *testing.T) {
	doc := `{"board_size":3,"players":3,"spaces":[` + home + route + enemy + `,{"x":-2,"y":2,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":2,"count":1}]}]}]}`
	fakeboard.UseConfig(t, shippedConfig, nil, map[string]string{"eliminated": doc})
	g := harness.NewGame(t, "custom:eliminated", 1, 3)
	p := g.Human(0)
	var orders []defs.OrderFromFrontend
	for _, u := range g.Self(p).Units {
		orders = append(orders, harness.Order(defs.OrderType_UnitDelete, []uint64{u.InternalID}, 0, true))
	}
	for _, b := range g.Self(p).Buildings {
		orders = append(orders, harness.Order(defs.OrderType_BuildingDelete, []uint64{b.InternalID}, 0, true))
	}
	g.Submit(p, orders...)
	g.EndTurn()
	if !g.Self(p).Eliminated || g.Base.GameEnded() {
		t.Fatal("setup did not leave an eliminated player and two survivors")
	}
	g.SubmitRejected(p, harness.OrderMove(nil, 1, 0), "Eliminated players cannot submit orders")
	turn := g.State(p).TurnNumber
	g.Submit(g.Human(1))
	g.Submit(g.Human(2))
	if g.State(p).TurnNumber != turn+1 {
		t.Fatal("eliminated player blocked turn resolution")
	}
}
