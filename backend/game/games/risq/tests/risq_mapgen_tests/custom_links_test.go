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
		"seam west to east": `{"from":[-3,0],"to":[3,0],"direction":3}`,
		"seam east to west": `{"from":[3,0],"to":[-3,0],"direction":0}`,
		"seam NE to NW":     `{"from":[-3,0],"to":[3,0],"direction":1,"to_direction":2}`,
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
	mustFail(t, connect(two, `{"from":[-3,0],"to":[3,0],"direction":3,"to_direction":6}`), 1, "not 0 to 5")
	mustFail(t, connect(two, `{"from":[-3,0],"to":[3,0]}`), 1, "needs a direction")
	mustFail(t, connect(`{"x":-3,"y":0},{"x":3,"y":0,"terrain":152}`, `{"from":[-3,0],"to":[3,0]}`), 1, "invalid connection")
	mustFail(t, connect(`{"x":0,"y":0},{"x":1,"y":0}`, `{"from":[0,0],"to":[1,0],"direction":0}`), 1, "already connected")
}

func TestSeamNeedsFreeEdgeOnBothSides(t *testing.T) {
	doc := `{"board_size":3,"players":1,"spaces":[{"x":-3,"y":0},{"x":3,"y":0},{"x":-3,"y":3}],"connections":[` +
		`{"from":[-3,0],"to":[3,0],"direction":3},{"from":[-3,0],"to":[-3,3],"direction":3}]}`
	mustFail(t, doc, 1, "already connected")
}

func TestDistanceCacheInvalidatesAfterRemovingSpace(t *testing.T) {
	board := mustGenerate(t, customDoc(1, "", `{"x":-1,"y":0},{"x":0,"y":0},{"x":1,"y":0}`), 1)
	left, right := board.Space(game_utils.Coordinate2D{X: -1}), board.Space(game_utils.Coordinate2D{X: 1})
	if board.Distance(left, right) != 2 || board.Distance(right, left) != 2 {
		t.Fatal("initial distance is not two hops")
	}
	board.RemoveSpace(board.Space(game_utils.Coordinate2D{}))
	if board.Distance(left, right) != fakeboard.Unreachable || board.Distance(right, left) != fakeboard.Unreachable {
		t.Error("removed space retained a cached path")
	}
}

func TestDistanceCacheInvalidatesAfterAllocation(t *testing.T) {
	board := fakeboard.New()
	board.Allocate(1)
	if board.Distance(board.Space(game_utils.Coordinate2D{X: -1}), board.Space(game_utils.Coordinate2D{X: 1})) != 2 {
		t.Fatal("initial distance is not two hops")
	}
	board.Allocate(2)
	left, right := board.Space(game_utils.Coordinate2D{X: -2}), board.Space(game_utils.Coordinate2D{X: 2})
	if board.Distance(left, right) != 4 || board.Distance(right, left) != 4 || board.Distance(left, left) != 0 {
		t.Error("new allocation retained stale distances or indices")
	}
}

func TestDistanceCacheInvalidatesAfterConnectingSpaces(t *testing.T) {
	board := linkedBoard(t, "")
	left, right := board.Space(game_utils.Coordinate2D{X: -3}), board.Space(game_utils.Coordinate2D{X: 3})
	if board.Distance(left, right) != fakeboard.Unreachable || board.Distance(right, left) != fakeboard.Unreachable {
		t.Fatal("initial spaces are connected")
	}
	if err := board.ConnectSpaces(left, right, 3, 0); err != nil {
		t.Fatal(err)
	}
	if board.Distance(left, right) != 1 || board.Distance(right, left) != 1 {
		t.Error("new seam retained cached unreachable distances")
	}
}
