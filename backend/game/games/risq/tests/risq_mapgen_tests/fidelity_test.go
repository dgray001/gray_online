package risq_mapgen_tests

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func engineStarts(mapName string, players int, seed int64) ([]risq.PlayerSnapshot, error) {
	base := game.CreateBaseGame(1, game.GameType_RISQ, aiSettings(players, seed, mapName), nil, nil)
	g, err := risq.CreateGame(base, make(chan game.PlayerAction, 4))
	if err != nil {
		return nil, err
	}
	g.StopAi()
	return g.Snapshot(), nil
}

// The fake board must agree with the real engine board, or every other test against it proves nothing
func checkFakeMatchesEngine(t *testing.T, mapName string, players int) {
	t.Helper()
	board, fakeErr := fakeboard.Generate(mapName, players, 1)
	snaps, engineErr := engineStarts(mapName, players, 1)
	if fmt.Sprint(fakeErr) != fmt.Sprint(engineErr) {
		t.Fatalf("fake error %v, engine error %v", fakeErr, engineErr)
	}
	for player, snap := range snaps {
		units, buildings, home := board.Tally(player)
		if !reflect.DeepEqual(units, snap.Units) || !reflect.DeepEqual(buildings, snap.Buildings) || home != snap.Home {
			t.Errorf("player %d: fake units %v buildings %v home %q; engine %v %v %q", player, units, buildings, home, snap.Units, snap.Buildings, snap.Home)
		}
		if bank, set := board.Banks[player]; set && (bank.Food != snap.Food || bank.Wood != snap.Wood || bank.Stone != snap.Stone || bank.Gold != snap.Gold) {
			t.Errorf("player %d: fake bank %+v, engine %v/%v/%v/%v", player, bank, snap.Food, snap.Wood, snap.Stone, snap.Gold)
		}
	}
}

func TestFakeBoardMatchesEngineOnScripts(t *testing.T) {
	t.Parallel()
	for _, name := range sweptScripts {
		for _, players := range sweptPlayerCounts(name)[:1] {
			t.Run(fmt.Sprintf("%s/p%d", name, players), func(t *testing.T) { checkFakeMatchesEngine(t, "script:"+name, players) })
		}
	}
}

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
