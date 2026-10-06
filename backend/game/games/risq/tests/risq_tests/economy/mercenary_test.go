package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

const mercenaryRules = `,"starting_techs":[4],"space_gold_income":0`

func TestMercenaryHiringChargesGoldAndPlacesUnit(t *testing.T) {
	g := productionGame(t, 1, 1, mercenaryRules)
	p := g.Human(0)
	g.Submit(p, harness.Order(defs.OrderType_BuyMercenary, nil, harness.BuildKey(11, 0, 0, 0, 0), false))
	g.EndTurn()
	state := g.Self(p)
	if len(state.Units) != 2 || state.Resources.Gold != 409 || state.Resources.Food != 300 || state.Resources.Wood != 300 {
		t.Fatalf("hiring: units %d, resources %+v; want two units and gold-only charge 91", len(state.Units), state.Resources)
	}
	hired := 0
	for _, u := range state.Units {
		if u.UnitID == 11 && u.Space == (harness.Coord{}) && u.Zone == (harness.Coord{}) {
			hired++
		}
	}
	if hired != 1 {
		t.Errorf("hired %d infantry in requested zone, want one", hired)
	}
	g.EndTurn()
	if state = g.Self(p); len(state.Units) != 2 || state.Resources.Gold != 409 {
		t.Error("hiring repeated on the following turn")
	}
}
