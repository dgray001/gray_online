package fiddlesticks

import (
	"fmt"
	"os"

	"github.com/dgray001/gray_online/game"
)

/**
Theory model 4 keeps track of all cards based on both definitively known information
and inferences based on playstyle. For example, if another player bids high the
model will increase the likelihood that the player has high cards
*/

type FiddlesticksAiModelTheory4 struct {
	// configurable model parameters
	// internal data for a given game
}

func createFiddlesticksAiModelTheory4(model_input map[string]string) FiddlesticksAiModelTheory4 {
	m := FiddlesticksAiModelTheory4{}
	for k, _ := range model_input {
		switch k {
		default:
			fmt.Fprintln(os.Stderr, "Unknown model input key for theory model 4:", k)
		}
	}
	return m
}

func (m FiddlesticksAiModelTheory4) ApplyUpdate(p *FiddlesticksPlayer, f *GameFiddlesticks, u *game.UpdateMessage) {
	switch u.Kind {
	case "deal-round":
		// TODO: implement
	case "bet":
		// TODO: implement
	case "play-card":
		// TODO: implement
	default:
		fmt.Fprintln(os.Stderr, "Unknown update kind for theory model 3:", u.Kind)
	}
}

func (m FiddlesticksAiModelTheory4) Bet(p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	return 0
}

func (m FiddlesticksAiModelTheory4) CardWeights(p *FiddlesticksPlayer, f *GameFiddlesticks, valid_cards []int) []float64 {
	weights := make([]float64, len(valid_cards))
	for i := range weights {
		weights[i] = 1
	}
	return weights
}
