package risq_mapgen_tests

import (
	"testing"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

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
	if err := board.ConnectSpaces(left, right, 3); err != nil {
		t.Fatal(err)
	}
	if board.Distance(left, right) != 1 || board.Distance(right, left) != 1 {
		t.Error("new seam retained cached unreachable distances")
	}
}
