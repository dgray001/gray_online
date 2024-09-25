package fiddlesticks

import (
	"fmt"
	"math"
	"os"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
)

/**
Theory model 2 is a model only accounting for each card individually
The hand as a whole and other players are not factored into this model
For playing cards, it doesn't account for future tricks, only how much it wants
	to win the current trick and whether a given card will win that trick
Calculations are based on a linearized heuristic of the value of a card
*/

type FiddlesticksAiModelTheory2 struct {
	trump_min     float64 // min bet value of a trump card
	non_trump_min uint8   // non trump card must be more than this to have bet value
	non_trump_max float64 // value of a non trump ace card
}

func createFiddlesticksAiModelTheory2(model_input map[string]string) FiddlesticksAiModelTheory2 {
	m := FiddlesticksAiModelTheory2{
		trump_min:     0.25,
		non_trump_min: 8,
		non_trump_max: 0.4,
	}
	for k, v := range model_input {
		switch k {
		case "trump_min":
			m.trump_min = util.ParseFloat(v)
		case "non_trump_min":
			m.non_trump_min = uint8(util.ParseInt(v))
		case "non_trump_max":
			m.non_trump_max = util.ParseFloat(v)
		default:
			fmt.Fprintln(os.Stderr, "Unknown model input key for theory model 2:", k)
		}
	}
	return m
}

func (m FiddlesticksAiModelTheory2) ApplyUpdate(p *FiddlesticksPlayer, f *GameFiddlesticks, u *game.UpdateMessage) {
}

func (m FiddlesticksAiModelTheory2) Bet(p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	bid := float64(0)
	trump_suit := f.trump.GetSuit()
	for _, card := range p.cards {
		bid += m.cardProbability(card, trump_suit)
	}
	return bid
}

func (m FiddlesticksAiModelTheory2) CardWeights(p *FiddlesticksPlayer, f *GameFiddlesticks, valid_cards []int) []float64 {
	weights := make([]float64, len(valid_cards))
	total_weight := float64(0)
	tricks_needed := p.bet - p.tricks
	tricks_left := len(p.cards) - len(p.cards_played)
	// how much the model should prioritize winning a trick
	tricks_needed_factor := float64(tricks_needed) / float64(tricks_left)
	current_winning_card, _ := f.winningTrickCard()
	for i, card_index := range valid_cards {
		card := p.cards[card_index]
		if !f.cardBeatsCard(card, current_winning_card) {
			weights[i] = m.calculateWeight(tricks_needed_factor, 0)
		} else if len(f.trick) == len(f.players)-1 {
			weights[i] = m.calculateWeight(tricks_needed_factor, 1)
		} else {
			prob_to_win := m.cardProbability(card, f.trump.GetSuit())
			prob_to_win += float64(len(f.trick)) / float64(len(f.players))
			if prob_to_win > 1 {
				prob_to_win = 1
			}
			weights[i] = m.calculateWeight(tricks_needed_factor, prob_to_win)
		}
		total_weight += weights[i]
	}
	return weights
}

func (m *FiddlesticksAiModelTheory2) cardProbability(card *game_utils.StandardCard, trump_suit uint8) float64 {
	if card.GetSuit() == trump_suit {
		return m.trump_min + (1-m.trump_min)*float64(card.GetNumber()-2)/12.0
	} else if card.GetNumber() > m.non_trump_min {
		return m.non_trump_max * float64(card.GetNumber()-m.non_trump_min) /
			float64(game_utils.StandardCardStatic.AceValue()-m.non_trump_min)
	}
	return 0
}

func (m *FiddlesticksAiModelTheory2) calculateWeight(tricks_needed_factor float64, probability_of_winning float64) float64 {
	return 1 - math.Abs(tricks_needed_factor-probability_of_winning)
}
