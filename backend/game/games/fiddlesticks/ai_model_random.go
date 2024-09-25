package fiddlesticks

import (
	"math/rand"

	"github.com/dgray001/gray_online/game"
)

/**
Random model just bets and plays cards randomly within what is legal
*/

type FiddlesticksAiModelRandom struct{}

func (m FiddlesticksAiModelRandom) ApplyUpdate(p *FiddlesticksPlayer, f *GameFiddlesticks, u *game.UpdateMessage) {
}

func (m FiddlesticksAiModelRandom) Bet(p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	return float64(f.round) * rand.Float64()
}

func (m FiddlesticksAiModelRandom) CardWeights(p *FiddlesticksPlayer, f *GameFiddlesticks, valid_cards []int) []float64 {
	weights := make([]float64, len(valid_cards))
	for i := range weights {
		weights[i] = 1
	}
	return weights
}
