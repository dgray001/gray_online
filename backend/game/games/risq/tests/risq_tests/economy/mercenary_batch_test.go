package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestMercenaryBatchReservesGoldAndPopulation(t *testing.T) {
	for name, c := range map[string]struct {
		extra, reason string
		hired         int
	}{"capacity": {"", "population capped", 4}, "affordability": {`,"unlimited_population":true`, "cannot afford mercenary", 5}} {
		t.Run(name, func(t *testing.T) {
			g := productionGame(t, 1, 1, mercenaryRules+c.extra)
			p := g.Human(0)
			orders := make([]defs.OrderFromFrontend, 6)
			for i := range orders {
				orders[i] = harness.Order(defs.OrderType_BuyMercenary, nil, harness.BuildKey(11, 0, 0, 0, 0), false)
			}
			g.Submit(p, orders...)
			g.EndTurn()
			state := g.Self(p)
			if len(state.Units) != 1+c.hired || state.Resources.Gold != float64(500-91*c.hired) {
				t.Errorf("batch: units %d, gold %v; want %d and %d", len(state.Units), state.Resources.Gold, 1+c.hired, 500-91*c.hired)
			}
			failures := state.Refusals()
			if len(failures) != 6-c.hired {
				t.Fatalf("failures %v, want %d", failures, 6-c.hired)
			}
			for _, reason := range failures {
				if reason != c.reason {
					t.Errorf("failure %q, want %q", reason, c.reason)
				}
			}
		})
	}
}
