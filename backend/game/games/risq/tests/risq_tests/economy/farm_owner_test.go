package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestFarmGatherAndRenewRequireOwnership(t *testing.T) {
	g := economyGame(t, `{"wood":120}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":3,"player":0},"units":[{"id":1,"player":0,"count":1},{"id":1,"player":1,"count":1}]}]}`, "")
	p, enemy := g.Human(0), g.Human(1)
	u := g.Self(enemy).Units[0].InternalID
	if g.State(enemy).Space(0, 0).Visibility < int(defs.VisibilityPoor) {
		t.Fatal("enemy cannot see the farm")
	}
	g.Submit(enemy, harness.OrderGather([]uint64{u}, 0, 0, 0, 0))
	g.EndTurn()
	if refusals := g.Self(enemy).Refusals(); len(refusals) != 1 || refusals[0] != "not receivable" {
		t.Fatalf("non-owner gather refusals %v", refusals)
	}
	if state := g.Self(enemy); state.Resources.Food != 0 || farmZone(g, p).Building.ResourcesLeft != 200 {
		t.Fatalf("rejected gather credited food or drained the farm: resources %+v", state.Resources)
	}
	id := fastExhaustFarm(g, p)
	g.Submit(enemy, harness.Order(defs.OrderType_UnitRenew, []uint64{u}, int64(id), false))
	g.EndTurn()
	state := g.Self(enemy)
	if state.Resources.Wood != 120 || len(state.Refusals()) != 1 || state.Refusals()[0] != "not receivable" {
		t.Errorf("non-owner renewal: resources %+v, refusals %v", state.Resources, state.Refusals())
	}
	if farmZone(g, p).Building.ResourcesLeft != 0 || g.Self(p).Resources.Wood != 120 {
		t.Error("rejected renewal changed the owner's farm or wood")
	}
}
