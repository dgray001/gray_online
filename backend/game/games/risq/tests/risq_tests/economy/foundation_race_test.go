package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestFoundationRaceRefundsLoser(t *testing.T) {
	winners := []int{}
	for _, slots := range [][2]int{{0, 1}, {1, 0}} {
		g := scenarioGame(t, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":0,"count":1},{"id":1,"player":1,"count":1}]}]},{"x":-1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":2,"player":0}}]},{"x":1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":2,"player":1}}]}`)
		for _, slot := range slots {
			p := g.Human(slot)
			g.Submit(p, housingOrders(g, p, 0)...)
		}
		winner, created := -1, 0
		for slot := 0; slot < 2; slot++ {
			state := g.Self(g.Human(slot))
			owns := false
			for _, b := range state.Buildings {
				if b.Space == (harness.Coord{}) {
					winner, owns = slot, true
					created++
				}
			}
			wantWood := 120.0
			if owns {
				wantWood -= 30
			}
			if state.Resources.Wood != wantWood || len(state.PlannedFoundations) != 0 {
				t.Errorf("slot %d: wood %v, foundations %v; want %v wood and no reservation", slot, state.Resources.Wood, state.PlannedFoundations, wantWood)
			}
			if !owns && len(state.ActiveOrders) != 0 {
				t.Errorf("loser retained orders %v", state.ActiveOrders)
			}
		}
		if created != 1 {
			t.Fatalf("created %d buildings at the contested zone, want one", created)
		}
		winners = append(winners, winner)
		g.EndTurn()
		if state := g.Self(g.Human(1 - winner)); state.Resources.Wood != 120 || len(state.Buildings) != 1 {
			t.Error("loser was charged again or created another building")
		}
	}
	if winners[0] != winners[1] {
		t.Errorf("submission order changed winner: %v", winners)
	}
}
