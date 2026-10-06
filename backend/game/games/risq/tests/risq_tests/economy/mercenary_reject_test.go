package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestMercenaryRestrictionsDoNotSpendGold(t *testing.T) {
	for name, c := range map[string]struct {
		workers, x    int
		extra, reason string
	}{
		"enemy territory": {1, -2, "", "space or zone not owned"},
		"unowned region":  {1, 0, `,"mercenaries_need_region":true,"regions":[{"name":"Split","spaces":[[0,0],[-2,0]]}]`, "region not owned"},
		"population cap":  {5, 0, "", "population capped"},
	} {
		t.Run(name, func(t *testing.T) {
			g := productionGame(t, c.workers, 1, mercenaryRules+c.extra)
			if name == "enemy territory" {
				g = economyGame(t, `{"gold":500}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":0},"units":[{"id":1,"player":0,"count":1}]}]},{"x":-2,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":1,"count":1}]}]}`, mercenaryRules)
				owner := g.State(g.Human(1)).Space(-2, 0).Ownership
				if owner == nil || *owner != g.PlayerID(g.Human(1)) {
					t.Fatal("destination is not enemy territory")
				}
			}
			p := g.Human(0)
			g.Submit(p, harness.Order(defs.OrderType_BuyMercenary, nil, harness.BuildKey(11, c.x, 0, 0, 0), false))
			g.EndTurn()
			state := g.Self(p)
			if len(state.Units) != c.workers || state.Resources.Gold != 500 {
				t.Errorf("rejected hire: units %d, gold %v", len(state.Units), state.Resources.Gold)
			}
			if failures := state.Refusals(); len(failures) != 1 || failures[0] != c.reason {
				t.Errorf("failures %v, want %q", failures, c.reason)
			}
		})
	}
}
