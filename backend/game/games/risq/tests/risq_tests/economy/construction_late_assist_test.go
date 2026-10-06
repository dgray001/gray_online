package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestLaterBuilderAssistsExistingConstruction(t *testing.T) {
	g := buildGame(t, 2)
	p := g.Human(0)
	workers := housingOrders(g, p, 0)
	g.Submit(p, workers[0])
	g.EndTurn()
	b := centerHousing(g, p)
	if !b.UnderConstruction || b.StaminaRemaining != 6 {
		t.Fatalf("setup did not leave unfinished housing: %+v", b)
	}
	order := g.Self(p).ActiveOrders[0].InternalID
	g.Submit(p, harness.Order(defs.OrderType_CancelOrder, nil, int64(order), false), workers[1])
	g.EndTurn()
	after := centerHousing(g, p)
	if after.InternalID != b.InternalID || after.UnderConstruction || g.Self(p).Resources.Wood != 90 || len(g.Self(p).Refusals()) != 0 {
		t.Errorf("later assistance did not finish the same housing with one payment: %+v", g.Self(p))
	}
}
