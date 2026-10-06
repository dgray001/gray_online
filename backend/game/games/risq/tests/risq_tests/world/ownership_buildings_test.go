package world

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestBuildingsOfDifferentPlayersLeaveASpaceUnowned(t *testing.T) {
	shared := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":2,"player":0}},{"x":1,"y":0,"building":{"id":2,"player":1}}]}`
	spaces := shared + "," + spaceWith(1, 0, grass, unit(villager, 0)+","+unit(villager, 1))
	g := startGame(t, mapDoc("", spaces))
	if got := g.State(g.Human(0)).Space(0, 0).Ownership; got == nil || *got != unowned {
		t.Errorf("a space holding both players' buildings has owner %v, want none", got)
	}
}

func TestRemovingTheLastBuildingHandsTheSpaceToTheSoldiers(t *testing.T) {
	g, p0 := ownershipGame(t)
	camp := g.Self(p0).Buildings[0].InternalID
	g.Submit(p0, harness.Order(defs.OrderType_BuildingDelete, []uint64{camp}, 0, false))
	g.EndTurn()
	if got := g.State(p0).Space(0, -1).Ownership; got == nil || *got != 1 {
		t.Errorf("(0,-1) owner %v after P0's camp was deleted, want P1 through its soldier", got)
	}
}
