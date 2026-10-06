package sims

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/aisim"
	"testing"
)

func assertSubmissions(t *testing.T, g *aisim.Game, slot, want int) {
	t.Helper()
	count := 0
	owned := make(map[uint64]bool)
	for _, unit := range g.Own(t, slot).Units {
		owned[unit.InternalID] = unit.PlayerID == slot
	}
	for _, action := range g.Trace {
		if action.Kind != "submit-orders" || action.Ai_id != int(g.Player(t, slot).GetAiId()) {
			continue
		}
		count++
		for _, order := range action.Action["orders"].([]defs.OrderFromFrontend) {
			for _, id := range order.Subjects {
				if !owned[id] {
					t.Fatalf("foreign subject: %+v", order)
				}
			}
		}
	}
	if count != want {
		t.Fatalf("slot %d submissions %d, want %d", slot, count, want)
	}
}
