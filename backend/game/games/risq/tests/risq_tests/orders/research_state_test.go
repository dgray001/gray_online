package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestResearchCannotBeSubmittedTwice(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	order := harness.Order(defs.OrderType_BuildingResearch, []uint64{buildingID(g, p, 11)}, 2, false)
	g.Submit(p, order)
	g.EndTurn()
	g.SubmitRejected(p, order, "Tech id 2 is already being researched")
	for range 2 {
		g.EndTurn()
	}
	if !g.Self(p).ResearchedTechs[2] {
		t.Fatal("setup research did not complete")
	}
	g.SubmitRejected(p, order, "Tech id 2 is already researched")
}
