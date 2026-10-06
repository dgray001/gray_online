package orders

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestRefundsPrecedeSpendingRegardlessOfBatchPosition(t *testing.T) {
	for _, kind := range []string{"production", "research", "foundation"} {
		for _, refundFirst := range []bool{false, true} {
			t.Run(kind+"/refund-first="+fmt.Sprint(refundFirst), func(t *testing.T) {
				g, p, refund, spend := refundOrders(t, kind)
				batch := []defs.OrderFromFrontend{spend, refund}
				if refundFirst {
					batch[0], batch[1] = batch[1], batch[0]
				}
				g.Submit(p, batch...)
				g.EndTurn()
				state := g.Self(p)
				if len(state.Refusals()) != 0 || len(state.ActiveOrders) != 1 {
					t.Fatalf("refund did not fund replacement: %+v", state)
				}
				if *state.Resources != (harness.Resources{}) {
					t.Fatalf("unexpected balance after refund and spend: %+v", state.Resources)
				}
				if state.ActiveOrders[0].OrderType != spend.Order_type {
					t.Fatal("refunded order survived instead of replacement")
				}
			})
		}
	}
}

func TestDuplicateCancellationRefundsOnlyOnce(t *testing.T) {
	g, p, refund, _ := refundOrders(t, "production")
	g.Submit(p, refund, refund)
	for range 2 {
		g.EndTurn()
		state := g.Self(p)
		if state.Resources.Food != 70 || state.Resources.Wood != 40 || len(state.ActiveOrders) != 0 || len(unitIDs(g, p, 12)) != 0 {
			t.Fatalf("duplicate cancellation refunded twice or left production active: %+v", state)
		}
	}
}
