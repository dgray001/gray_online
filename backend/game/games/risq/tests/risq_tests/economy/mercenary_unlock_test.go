package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func TestResearchUnlocksMercenaryContracts(t *testing.T) {
	doc := `{"board_size":2,"players":2,"space_gold_income":0,"starting_bank":{"wood":300,"gold":500},"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":23,"player":0},"units":[{"id":1,"player":0,"count":1}]}]},{"x":-2,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":1,"count":1}]}]}]}`
	fakeboard.UseConfig(t, shippedConfig, nil, map[string]string{"contracts": doc})
	g := harness.NewGame(t, "custom:contracts", 1, 2)
	p := g.Human(0)
	if len(g.Self(p).AvailableMercenaries) != 0 {
		t.Fatal("mercenaries available before contracts research")
	}
	b := g.Self(p).Buildings[0]
	g.Submit(p, harness.Order(defs.OrderType_BuildingResearch, []uint64{b.InternalID}, 4, false))
	g.EndTurn()
	state := g.Self(p)
	if !state.ResearchedTechs[4] || state.Resources.Wood != 150 || len(state.AvailableMercenaries) != 2 {
		t.Fatalf("contracts research did not unlock both infantry with one payment: %+v", state)
	}
	if state.AvailableMercenaries[0].ID != 11 || state.AvailableMercenaries[1].ID != 12 {
		t.Fatalf("unlocked wrong mercenaries: %v", state.AvailableMercenaries)
	}
	g.Submit(p, harness.Order(defs.OrderType_BuyMercenary, nil, harness.BuildKey(12, 0, 0, 0, 0), false))
	g.EndTurn()
	state = g.Self(p)
	if len(state.Units) != 2 || state.Resources.Gold != 357 || state.Resources.Wood != 150 {
		t.Errorf("unlocked hiring: units %d, resources %+v", len(state.Units), state.Resources)
	}
}
