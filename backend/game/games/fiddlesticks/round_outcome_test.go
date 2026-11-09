package fiddlesticks

import (
	"github.com/dgray001/gray_online/game"
	"testing"
)

func TestRoundObserverSeesTerminalResultsBeforeReset(t *testing.T) {
	base := game.CreateBaseGame(1, game.GameType_FIDDLESTICKS, nil, nil, nil)
	f := InitializeGame(base)
	f.AddInternalAiPlayer(4, nil)
	f.AddInternalAiPlayer(4, nil)
	f.SetSettings(1, 2, 0, 5, 1)
	f.SetDealSeed(1)
	f.SetActionSeed(2)
	rounds := []uint8{}
	f.SetRoundObserver(func(outcomes []RoundOutcome) {
		rounds = append(rounds, outcomes[0].Round)
		for _, outcome := range outcomes {
			p := f.players[outcome.Seat]
			if p.bet != outcome.Bid || p.tricks != outcome.Tricks {
				t.Fatal("round reset before reward")
			}
			points := uint32(0)
			if outcome.Bid == outcome.Tricks {
				points = uint32(outcome.Bid) * 5
			}
			if outcome.Points != points {
				t.Fatal(outcome)
			}
		}
	})
	base.StartGame()
	f.StartAiGame(false)
	for !f.ExecuteAiTurn(false) {
	}
	if len(rounds) != 3 || rounds[0] != 1 || rounds[1] != 2 || rounds[2] != 1 {
		t.Fatal("missing final round", rounds)
	}
}
