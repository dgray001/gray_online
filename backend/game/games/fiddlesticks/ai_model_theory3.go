package fiddlesticks

import (
	"fmt"
	"os"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
)

/**
Theory model 3 keeps track of all cards based on definitely known information
Theory model 3 makes no inferences based on playstyle like another player's bid
*/

type FiddlesticksAiModelTheory3 struct {
	// configurable model parameters
	// internal data for a given game
	cards           map[uint16]*CardLocationProbability
	base_prob_kitty float64
	base_prob_hand  float64
}

func createFiddlesticksAiModelTheory3(model_input map[string]string) FiddlesticksAiModelTheory3 {
	m := FiddlesticksAiModelTheory3{
		cards:           make(map[uint16]*CardLocationProbability),
		base_prob_kitty: 0,
		base_prob_hand:  0,
	}
	for _, card := range game_utils.CreateStandardDeck().GetCards() {
		if card == nil {
			continue
		}
		probabilities := &CardLocationProbability{
			played: false,
			hand:   false,
			trump:  false,
			kitty:  0,
			player: make(map[int]float64),
		}
		m.cards[card.Hash()] = probabilities
	}
	for k := range model_input {
		switch k {
		default:
			fmt.Fprintln(os.Stderr, "Unknown model input key for theory model 3:", k)
		}
	}
	return m
}

func (m FiddlesticksAiModelTheory3) ApplyUpdate(p *FiddlesticksPlayer, f *GameFiddlesticks, u *game.UpdateMessage) {
	fmt.Println("Applying update", u)
	switch u.Kind {
	case "deal-round":
		hand_hashes := make(map[uint16]bool)
		for _, card := range p.cards {
			hand_hashes[card.Hash()] = true
		}
		cards_unknown := 52 - 1 - int(f.round)
		cards_in_kitty := 52 - 1 - len(f.players)*int(f.round)
		m.base_prob_kitty = float64(cards_in_kitty) / float64(cards_unknown)
		m.base_prob_hand = float64(f.round) / float64(cards_unknown)
		for hash, probabilities := range m.cards {
			probabilities.reset(f.players, p.player.Player_id)
			if hash == f.trump.Hash() {
				probabilities.isTrump()
			} else if util.MapContains(hand_hashes, hash) {
				probabilities.inHand()
			} else {
				probabilities.unknown(m.base_prob_kitty, m.base_prob_hand)
			}
		}
	case "bet":
		// TODO: implement
	case "play-card":
		// TODO: implement
	default:
		fmt.Fprintln(os.Stderr, "Unknown update kind for theory model 3:", u.Kind)
	}
}

func (m FiddlesticksAiModelTheory3) Bet(p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	return 0
}

func (m FiddlesticksAiModelTheory3) CardWeights(p *FiddlesticksPlayer, f *GameFiddlesticks, valid_cards []int) []float64 {
	weights := make([]float64, len(valid_cards))
	for i := range weights {
		weights[i] = 1
	}
	return weights
}
