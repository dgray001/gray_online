package world

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestRemovedSpacesDoNotExist(t *testing.T) {
	g := startGame(t, mapDoc("", spaceWith(0, 0, grass, unit(villager, 0))+","+spaceWith(1, 0, grass, unit(villager, 1))+","+spaceWith(3, 0, grass, "")))
	state := g.State(0)
	if state.SpaceCount() != 3 {
		t.Errorf("%d spaces in the payload, want only the 3 listed", state.SpaceCount())
	}
	if state.Space(2, 0) != nil || state.Space(0, 1) != nil {
		t.Error("an unlisted space appears in the payload")
	}
}

func TestLinksAndSeamsAreExportedToClients(t *testing.T) {
	spaces := spaceWith(-3, 0, grass, unit(villager, 0)) + "," + spaceWith(3, 0, grass, unit(villager, 1))
	connections := `,"connections":[{"from":[-3,0],"to":[3,0],"direction":3}]`
	state := startGame(t, mapDoc(connections, spaces)).State(0)
	if len(state.SpaceLinks) != 1 {
		t.Fatalf("%d links exported, want exactly 1 seam", len(state.SpaceLinks))
	}
	want := harness.SpaceLink{From: harness.Coord{X: -3}, To: harness.Coord{X: 3}, Direction: 3}
	if link := state.SpaceLinks[0]; link != want {
		t.Errorf("seam %+v, want %+v: direction 3 leaves From westward", link, want)
	}
}

func TestOrdinaryBoardsExportNoLinks(t *testing.T) {
	state := startGame(t, mapDoc("", spaceWith(0, 0, grass, unit(villager, 0))+","+spaceWith(1, 0, grass, unit(villager, 1)))).State(0)
	if len(state.SpaceLinks) != 0 {
		t.Errorf("links = %v, want none", state.SpaceLinks)
	}
}
