package fiddlesticks

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"math"
	"testing"
)

func TestOneCardOddsAccountForLeading(t *testing.T) {
	f := InitializeGame(nil)
	for i := 0; i < 4; i++ {
		f.AddInternalAiPlayer(0, nil)
	}
	f.round, f.dealer = 1, 3
	f.trump = game_utils.StandardCardStatic.CreateCard(1, 14)
	f.players[0].cards = []*game_utils.StandardCard{game_utils.StandardCardStatic.CreateCard(2, 5)}
	f.players[1].cards = f.players[0].cards
	lead, total, beat := f.oneCardOdds(f.players[0])
	follow, _, _ := f.oneCardOdds(f.players[1])
	if total != 50 || beat != 21 || math.Abs(lead-3654.0/19600) > 1e-12 || math.Abs(follow-0.019285714285714285) > 1e-12 {
		t.Fatal(lead, follow, total, beat)
	}
}
