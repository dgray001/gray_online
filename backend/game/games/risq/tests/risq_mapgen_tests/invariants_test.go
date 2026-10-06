package risq_mapgen_tests

import (
	"reflect"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

// What every generated map must satisfy regardless of its script
func checkBoardInvariants(t *testing.T, board *fakeboard.Board, players int) {
	t.Helper()
	if len(board.Violations) > 0 {
		t.Errorf("steps broke placement rules: %v", board.Violations)
	}
	if groups := board.PassableComponents(); groups != 1 {
		t.Errorf("passable spaces form %d groups, want 1", groups)
	}
	owners := map[string]int{}
	spaces := board.PlayerSpaces()
	for player, held := range spaces {
		if player < 0 || player >= players {
			t.Errorf("placement for player %d with only %d players", player, players)
		}
		for coord := range held {
			key := coord.ToString()
			if other, shared := owners[key]; shared && other != player {
				t.Errorf("space %s holds players %d and %d", key, other, player)
			}
			owners[key] = player
		}
	}
	if len(spaces) != players {
		t.Errorf("%d players placed, want %d", len(spaces), players)
	}
	units0, buildings0, _ := board.Tally(0)
	for player := 1; player < players; player++ {
		units, buildings, _ := board.Tally(player)
		if !reflect.DeepEqual(units, units0) || !reflect.DeepEqual(buildings, buildings0) {
			t.Errorf("player %d kit %v %v differs from player 0 kit %v %v", player, units, buildings, units0, buildings0)
		}
	}
}
