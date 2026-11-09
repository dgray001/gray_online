package fiddlesticks

import (
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
	"testing"
)

func TestPublicHistoryPreservesActorsAndResetsRoundVoids(t *testing.T) {
	b := game.CreateBaseGame(1, game.GameType_FIDDLESTICKS, nil, nil, nil)
	f := InitializeGame(b)
	f.AddInternalAiPlayer(0, nil)
	f.AddInternalAiPlayer(0, nil)
	f.SetSettings(1, 2, 10, 1, 1)
	b.StartGame()
	f.StartAiGame(false)
	f.players[0].cards = []*game_utils.StandardCard{game_utils.StandardCardStatic.CreateCard(1, 2)}
	f.players[1].cards = []*game_utils.StandardCard{game_utils.StandardCardStatic.CreateCard(4, 2)}
	f.trump = game_utils.StandardCardStatic.CreateCard(3, 14)
	f.executeBet(f.players[1].player, 1, false, nil)
	f.executeBet(f.players[0].player, 0, false, nil)
	sawVoid := false
	f.SetRoundObserver(func(_ []RoundOutcome) { sawVoid = f.known_voids[0][3] && len(f.round_history) == 2 })
	f.executePlayCard(f.players[1].player, 0, false, nil)
	f.executePlayCard(f.players[0].player, 0, false, nil)
	if !sawVoid || f.known_voids[0][3] || len(f.round_history) != 0 || len(f.public_history) != 4 {
		t.Fatal("public history or round reset incorrect")
	}
	if f.public_history[3].Seat != 0 || f.public_history[3].Trick != 1 || f.public_history[3].Order != 1 || f.public_profiles[0].Hands != 1 || f.public_profiles[0].Exact != 1 {
		t.Fatal("actor, timing, or profile missing")
	}
}
