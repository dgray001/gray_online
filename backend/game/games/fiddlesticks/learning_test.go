package fiddlesticks

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"testing"
)

func TestLearningStateExcludesHiddenOpponentCards(t *testing.T) {
	f := InitializeGame(nil)
	f.AddInternalAiPlayer(0, nil)
	f.AddInternalAiPlayer(0, nil)
	f.SetSettings(1, 1, 12, 3, 1)
	deck := game_utils.CreateStandardDeck().GetCards()
	f.players[0].cards = deck[:2]
	f.players[1].cards = deck[2:4]
	f.players[1].cards_played = []int{0}
	state := f.LearningState(f.players[0])
	if state.RoundPoints != 12 || state.TrickPoints != 3 {
		t.Fatal("scoring settings missing from learning state")
	}
	if len(state.Hand) != 2 || state.Hand[0] != deck[0] || len(state.Seen) != 1 || state.Seen[0] != deck[2] {
		t.Fatal("hidden card leaked or public card missing")
	}
}
