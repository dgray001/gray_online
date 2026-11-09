package neural

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
)

const LegacyInputs = 370
const Inputs = 740
const FeatureVersion = 2
const Outputs = 79

func cardSlot(card *game_utils.StandardCard) int {
	return int(card.GetSuit()-1)*13 + int(card.GetNumber()-2)
}

func Features(state fiddlesticks.LearningState, betting bool) []float64 {
	input := make([]float64, Inputs)
	copy(input, legacyFeatures(state, betting))
	historyFeatures(input, state)
	progressFeatures(input, state)
	return input
}

func legacyFeatures(state fiddlesticks.LearningState, betting bool) []float64 {
	input := make([]float64, LegacyInputs)
	for _, card := range state.Hand {
		input[cardSlot(card)] = 1
	}
	for _, card := range state.Seen {
		input[cardSlot(card)] = 0
		input[52+cardSlot(card)] = 1
	}
	for i, card := range state.Trick {
		input[104+cardSlot(card)] = float64(i+1) / float64(len(state.Bets))
	}
	input[156+int(state.Trump.GetSuit()-1)] = 1
	return publicFeatures(input, state, betting)
}
