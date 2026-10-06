package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

func TestDepletionConservesResources(t *testing.T) {
	g := gatherGame(t, 11, false)
	p := g.Human(0)
	config := defs.ResourceConfigs[11]
	before := balance(g, p, config.Category)
	submitGather(g, p)
	deplete(g, p)
	after := balance(g, p, config.Category)
	if got := after - before; got != config.Starting_resources {
		t.Errorf("gained %v wood, want the entire pool of %v", got, config.Starting_resources)
	}
	g.EndTurn()
	if got := balance(g, p, config.Category); got != after {
		t.Errorf("wood changed after depletion: %v to %v", after, got)
	}
}
