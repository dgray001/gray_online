package sims

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"testing"
)

func TestThreePlayersKeepDecisionsSeparate(t *testing.T) {
	food := once(`{"action":"gather","category":"food","max":1}`)
	wood := once(`{"action":"gather","category":"wood","max":1}`)
	spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":0,"count":1}],"resource":1},{"x":1,"y":0,"building":{"id":2,"player":0}}]},` + remote + `,{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":2,"count":1}],"resource":11},{"x":1,"y":0,"building":{"id":2,"player":2}}]}`
	g := fixture(t, `{}`, spaces, "", food, emptyModel, wood)
	g.Run(t, 2)
	assertOrderCount(t, g, 0, defs.OrderType_UnitGather, 1)
	assertOrderCount(t, g, 2, defs.OrderType_UnitGather, 1)
	if g.Own(t, 0).Resources.Food <= 0 || g.Own(t, 0).Resources.Wood != 0 || g.Own(t, 2).Resources.Wood <= 0 || g.Own(t, 2).Resources.Food != 0 {
		t.Fatal("cross-player resource accounting")
	}
	for slot := range 3 {
		assertSubmissions(t, g, slot, 2)
	}
}
