package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestPopulationSupportIsClampedToGameMaximum(t *testing.T) {
	g := economyGame(t, `{}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":23,"player":0},"units":[{"id":1,"player":0,"count":1}]}]},{"x":1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":23,"player":0}}]},{"x":2,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":23,"player":0}}]},`+remoteVillager, "")
	p := g.Human(0)
	if g.Self(p).PopulationLimit != 100 {
		t.Fatal("150 support did not clamp to the 100-unit population limit")
	}
	for _, want := range []int{100, 50} {
		g.Submit(p, harness.Order(defs.OrderType_BuildingDelete, []uint64{g.Self(p).Buildings[0].InternalID}, 0, false))
		g.EndTurn()
		if g.Self(p).PopulationLimit != want {
			t.Errorf("remaining support produced limit %d, want %d", g.Self(p).PopulationLimit, want)
		}
	}
}
