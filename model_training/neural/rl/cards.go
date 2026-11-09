package rl

import (
	"fiddlesticks.live/neural"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
)

func (a *Agent) CardWeights(p *fiddlesticks.FiddlesticksPlayer, f *fiddlesticks.GameFiddlesticks, valid []int) []float64 {
	state := f.LearningState(p)
	legal := make([]int, len(valid))
	for i, index := range valid {
		card := state.Hand[index]
		legal[i] = 27 + int(card.GetSuit()-1)*13 + int(card.GetNumber()-2)
	}
	choice := a.choose(neural.FeaturesFor(a.Network, state, false), legal)
	weights := make([]float64, len(valid))
	for i, action := range legal {
		if action == choice {
			weights[i] = 1
		}
	}
	return weights
}
