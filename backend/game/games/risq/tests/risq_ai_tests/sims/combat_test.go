package sims

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"testing"
)

func TestExplicitAttackAndBehaviorReachEngine(t *testing.T) {
	model := once(`{"action":"set_unit_behavior","stance":"passive","attack_back":false},{"action":"set_building_behavior","auto_attack":false},{"action":"attack","target":"economic","max":1}`)
	spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":0},"units":[{"id":11,"player":0,"count":1},{"id":1,"player":1,"count":1}]}]},` + remote
	g := fixture(t, `{}`, spaces, "", model, emptyModel)
	g.Run(t, 1)
	assertOrderCount(t, g, 0, defs.OrderType_UnitAttackUnit, 1)
	if len(g.Own(t, 1).Units) != 1 {
		t.Fatalf("enemy survived: %+v", g.Own(t, 1).Units)
	}
	if g.Risq.Results().Players[0].Kills != 1 {
		t.Fatalf("results=%+v", g.Risq.Results())
	}
	assertBehaviorSequence(t, g, 0)
}
