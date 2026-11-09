package goldset

import (
	"fmt"
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
)

func prepareFoundation(spec FoundationCase, seed int64) (*fiddlesticks.GameFiddlesticks, error) {
	hands, err := completeFoundationHands(spec, seed)
	if err != nil {
		return nil, err
	}
	base := game.CreateBaseGame(1, game.GameType_FIDDLESTICKS, nil, nil, nil)
	f := fiddlesticks.InitializeGame(base)
	for seat := 0; seat < spec.Players; seat++ {
		f.AddInternalAiPlayer(4, nil)
	}
	hand := uint8(len(spec.Hand))
	f.SetSettings(hand, hand, spec.RoundPoints, spec.TrickPoints, 1)
	f.SetActionSeed(seed + 7919)
	if err := f.StartLearningDeal(hands, game_utils.StandardCardStatic.CreateCard(spec.Trump.Suit, spec.Trump.Number), spec.Players-1); err != nil {
		return nil, err
	}
	for f.IsBetting() {
		if spec.Kind == "bet" && f.LearningTurn() == spec.Seat {
			break
		}
		if err := f.ApplyLearningAction(int(spec.Bids[f.LearningTurn()])); err != nil {
			return nil, err
		}
	}
	for _, play := range spec.Plays {
		if f.LearningTurn() != play.Seat {
			return nil, fmt.Errorf("script actor mismatch in %s", spec.Name)
		}
		if err := f.ApplyLearningAction(27 + int(play.Card.Suit-1)*13 + int(play.Card.Number-2)); err != nil {
			return nil, err
		}
	}
	if f.LearningTurn() != spec.Seat || f.IsBetting() != (spec.Kind == "bet") || f.GetBase().GameEnded() {
		return nil, fmt.Errorf("wrong target decision in %s", spec.Name)
	}
	return f, nil
}
