package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

func TestContestedDepletionSplitsFairly(t *testing.T) {
	var results [][2]float64
	for _, slots := range [][2]int{{0, 1}, {1, 0}} {
		g := gatherGame(t, 11, true)
		p0, p1 := g.Human(0), g.Human(1)
		category := defs.ResourceConfigs[11].Category
		before := [2]float64{balance(g, p0, category), balance(g, p1, category)}
		for _, slot := range slots {
			submitGather(g, g.Human(slot))
		}
		deplete(g, p0)
		gains := [2]float64{balance(g, p0, category) - before[0], balance(g, p1, category) - before[1]}
		if gains[0] != gains[1] {
			t.Errorf("submission order %v: unequal shares %v", slots, gains)
		}
		if total := gains[0] + gains[1]; total != defs.ResourceConfigs[11].Starting_resources {
			t.Errorf("submission order %v: total gathered %v, want %v", slots, total, defs.ResourceConfigs[11].Starting_resources)
		}
		results = append(results, gains)
	}
	if results[0] != results[1] {
		t.Errorf("submission order changed gains: %v", results)
	}
}
