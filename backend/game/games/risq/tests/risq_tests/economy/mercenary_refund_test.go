package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestMercenaryPlacementFailureRefundsReservation(t *testing.T) {
	for name, c := range map[string]struct {
		building uint32
		reason   string
	}{"ownership": {3, "zone lost ownership before placement"}, "population": {2, "population capped before placement"}} {
		t.Run(name, func(t *testing.T) {
			g := economyGame(t, `{"gold":500}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":3,"player":0},"units":[{"id":1,"player":0,"count":1}]}]},{"x":1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":2,"player":0}}]},`+remoteVillager, mercenaryRules)
			p := g.Human(0)
			var target uint64
			for _, b := range g.Self(p).Buildings {
				if b.BuildingID == c.building {
					target = b.InternalID
				}
			}
			g.Submit(p, harness.Order(defs.OrderType_BuyMercenary, nil, harness.BuildKey(11, 0, 0, 0, 0), false), harness.Order(defs.OrderType_BuildingDelete, []uint64{target}, 0, false))
			g.EndTurn()
			state := g.Self(p)
			if state.Resources.Gold != 500 || len(state.Units) != 1 || len(state.Refusals()) != 1 || state.Refusals()[0] != c.reason {
				t.Fatalf("placement did not refund without spawning: %+v, gold %v", state, state.Resources.Gold)
			}
			g.EndTurn()
			if g.Self(p).Resources.Gold != 500 || len(g.Self(p).Units) != 1 {
				t.Error("placement refund or spawn repeated")
			}
		})
	}
}
