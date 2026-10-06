package economy

import (
	"math"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

func TestGatherCreditsTheCorrectCategory(t *testing.T) {
	for name, id := range map[string]uint32{"food": 1, "wood": 11, "stone": 41} {
		t.Run(name, func(t *testing.T) {
			g := gatherGame(t, id, false)
			p := g.Human(0)
			config := defs.ResourceConfigs[id]
			before := *g.Self(p).Resources
			want := float64(g.Self(p).Units[0].CurrentStamina) * float64(config.Base_gather_speed) / 10
			submitGather(g, p)
			g.EndTurn()
			initial := []float64{before.Food, before.Wood, before.Stone}
			for i, category := range []defs.RisqResourceCategory{defs.RisqResourceCategory_FOOD, defs.RisqResourceCategory_WOOD, defs.RisqResourceCategory_STONE} {
				expected := 0.0
				if category == config.Category {
					expected = want
				}
				if got := balance(g, p, category) - initial[i]; math.Abs(got-expected) > 0.00001 {
					t.Errorf("category %d gained %v, want %v", category, got, expected)
				}
			}
			resources := g.State(p).Space(0, 0).Resources
			if len(resources) != 1 || math.Abs(resources[0].ResourcesLeft-(config.Starting_resources-want)) > 0.00001 {
				t.Errorf("remaining resources %+v, want %v", resources, config.Starting_resources-want)
			}
		})
	}
}
