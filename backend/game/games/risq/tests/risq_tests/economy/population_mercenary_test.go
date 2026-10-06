package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestMercenaryReservationBlocksProductionAtCap(t *testing.T) {
	g := productionGame(t, 4, 1, mercenaryRules)
	p := g.Human(0)
	g.Submit(p, harness.OrderProduce([]uint64{centers(g, p)[0].InternalID}, 1), harness.Order(defs.OrderType_BuyMercenary, nil, harness.BuildKey(11, 0, 0, 0, 0), false))
	g.EndTurn()
	state := g.Self(p)
	if len(state.Units) != 5 || state.Resources.Food != 250 || state.Resources.Gold != 409 || len(centers(g, p)[0].ProductionQueue) != 1 || centers(g, p)[0].ProductionQueue[0].StaminaRemaining != 10 {
		t.Fatalf("production ignored reserved slot: %+v", state)
	}
}
