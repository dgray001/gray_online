package risq_mapgen_tests

import (
	"testing"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

// Two spaces six steps apart (-3,0) and (3,0), joined only by the given connections
func linkedBoard(t *testing.T, connections string) *fakeboard.Board {
	t.Helper()
	doc := `{"board_size":3,"players":1,"spaces":[{"x":-3,"y":0},{"x":3,"y":0}],"connections":[` + connections + `]}`
	return mustGenerate(t, doc, 1)
}

func between(board *fakeboard.Board) int {
	return board.Distance(board.Space(game_utils.Coordinate2D{X: -3}), board.Space(game_utils.Coordinate2D{X: 3}))
}

func TestCustomConnectionsJoinDistantSpaces(t *testing.T) {
	cases := map[string]string{
		"center link":       `{"from":[-3,0],"to":[3,0]}`,
		"seam west to east": `{"from":[-3,0],"to":[3,0],"direction":3}`,
		"seam east to west": `{"from":[3,0],"to":[-3,0],"direction":0}`,
	}
	for name, connections := range cases {
		if got := between(linkedBoard(t, connections)); got != 1 {
			t.Errorf("%s: distance %d, want 1", name, got)
		}
	}
	if got := between(linkedBoard(t, "")); got != fakeboard.Unreachable {
		t.Errorf("unconnected spaces: distance %d, want %d", got, fakeboard.Unreachable)
	}
}

func TestCustomConnectionsRejectBadEnds(t *testing.T) {
	connect := func(spaces string, body string) string {
		return `{"board_size":3,"players":1,"spaces":[` + spaces + `],"connections":[` + body + `]}`
	}
	two := `{"x":-3,"y":0},{"x":3,"y":0}`
	mustFail(t, connect(two, `{"from":[-3,0],"to":[-3,0]}`), 1, "invalid connection")
	mustFail(t, connect(two, `{"from":[-3,0],"to":[9,9]}`), 1, "invalid connection")
	mustFail(t, connect(two, `{"from":[-3,0],"to":[3,0],"direction":6}`), 1, "not 0 to 5")
	mustFail(t, connect(`{"x":-3,"y":0},{"x":3,"y":0,"terrain":152}`, `{"from":[-3,0],"to":[3,0]}`), 1, "invalid connection")
	mustFail(t, connect(`{"x":0,"y":0},{"x":1,"y":0}`, `{"from":[0,0],"to":[1,0],"direction":0}`), 1, "already connected")
}

func TestSeamNeedsFreeEdgeOnBothSides(t *testing.T) {
	doc := `{"board_size":3,"players":1,"spaces":[{"x":-3,"y":0},{"x":3,"y":0},{"x":-3,"y":3}],"connections":[` +
		`{"from":[-3,0],"to":[3,0],"direction":3},{"from":[-3,0],"to":[-3,3],"direction":3}]}`
	mustFail(t, doc, 1, "already connected")
}
