package economy

import (
	"fmt"
	"testing"
)

func TestTerrainChangesConstructionSpeed(t *testing.T) {
	for name, c := range map[string]struct{ terrain, remaining, turns int }{"grass": {1, 6, 2}, "hills": {51, 8, 3}} {
		t.Run(name, func(t *testing.T) {
			spaces := fmt.Sprintf(`{"x":0,"y":0,"terrain":%d,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":0,"count":1}]}]},{"x":1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":2,"player":0}}]},{"x":-2,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":1,"count":1}]}]}`, c.terrain)
			g := scenarioGame(t, spaces)
			p := g.Human(0)
			g.Submit(p, housingOrders(g, p, 0)...)
			g.EndTurn()
			b := centerHousing(g, p)
			if !b.UnderConstruction || b.StaminaRemaining != c.remaining {
				t.Errorf("first-turn housing %+v, want %d stamina remaining", b, c.remaining)
			}
			turns := 1
			for centerHousing(g, p).UnderConstruction && turns < 6 {
				g.EndTurn()
				turns++
			}
			finished := centerHousing(g, p)
			if turns != c.turns || finished.UnderConstruction || finished.InternalID != b.InternalID {
				t.Errorf("housing completed after %d turns: %+v, want %d turns and same building", turns, finished, c.turns)
			}
			if got := g.Self(p).Resources.Wood; got != 90 {
				t.Errorf("wood %v, want one 30-wood payment", got)
			}
		})
	}
}
