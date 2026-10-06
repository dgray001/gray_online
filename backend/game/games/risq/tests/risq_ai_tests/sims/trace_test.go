package sims

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/aisim"
)

func assertOrderCount(t *testing.T, g *aisim.Game, slot int, kind defs.OrderType, want int) {
	t.Helper()
	count := 0
	for _, action := range g.Trace {
		if action.Kind != "submit-orders" || action.Ai_id != int(g.Player(t, slot).GetAiId()) {
			continue
		}
		for _, order := range action.Action["orders"].([]defs.OrderFromFrontend) {
			if order.Player_id != slot {
				t.Fatalf("wrong player: %+v", order)
			}
			if order.Order_type == uint8(kind) {
				count++
			}
		}
	}
	if count != want {
		t.Fatalf("slot %d kind %d count %d, want %d; trace=%+v", slot, kind, count, want, g.Trace)
	}
}
