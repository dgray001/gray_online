package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestUnaffordableRenewalLeavesFarmDepleted(t *testing.T) {
	g := farmGame(t, 59, 1)
	p := g.Human(0)
	id := fastExhaustFarm(g, p)
	g.Submit(p, harness.Order(defs.OrderType_UnitRenew, []uint64{g.Self(p).Units[0].InternalID}, int64(id), false))
	g.EndTurn()
	if state := g.Self(p); state.Resources.Wood != 59 || farmZone(g, p).Building.ResourcesLeft != 0 || len(state.ActiveOrders) != 0 || len(state.Refusals()) != 1 || state.Refusals()[0] != "cannot afford renew" {
		t.Errorf("unaffordable renewal changed farm or balance: %+v", state)
	}
}
