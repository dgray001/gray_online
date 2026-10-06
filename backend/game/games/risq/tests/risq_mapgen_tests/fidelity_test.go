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
	base := game.CreateBaseGame(1, game.GameType_RISQ, aiSettings(players, seed, mapName))
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
	for _, name := range sweptScripts {
		for _, players := range sweptPlayerCounts(name)[:2] {
			t.Run(fmt.Sprintf("%s/p%d", name, players), func(t *testing.T) { checkFakeMatchesEngine(t, "script:"+name, players) })
		}
	}
}
