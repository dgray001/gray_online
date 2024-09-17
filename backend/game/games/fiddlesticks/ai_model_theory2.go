package fiddlesticks

import (
	"math"

	"github.com/dgray001/gray_online/game/game_utils"
)

/**
Theory model 2 is a model only accounting for each card individually
The hand as a whole and other players are not factored into this model
For playing cards, it doesn't account for future tricks, only how much it wants
	to win the current trick and whether a given card will win that trick
Calculations are based on a linearized heuristic of the value of a card
*/

type FiddlesticksAiModelTheory2 struct {
	trump_value_base    float64 // min bet value of a trump card
	non_trump_min_value uint8   // non trump card must be more than this to have bet value
	non_trump_ace_value float64 // value of a non trump ace card
}

func createFiddlesticksAiModelTheory2() FiddlesticksAiModelTheory2 {
	return FiddlesticksAiModelTheory2{
		trump_value_base:    0.3,
		non_trump_min_value: 5,
		non_trump_ace_value: 0.5,
	}
}

func (m FiddlesticksAiModelTheory2) Bet(p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	bid := float64(0)
	trump_suit := f.trump.GetSuit()
	for _, card := range p.cards {
		if card.GetSuit() == trump_suit {
			bid += m.trump_value_base + (1-m.trump_value_base)*(float64(card.GetNumber())-2)/12.0
		} else if card.GetNumber() > m.non_trump_min_value {
			bid += m.non_trump_ace_value * float64(card.GetNumber()-m.non_trump_min_value) /
				float64(game_utils.StandardCardStatic.AceValue()-m.non_trump_min_value)
		}
	}
	return bid
}

func (m FiddlesticksAiModelTheory2) PlayCard(p *FiddlesticksPlayer, f *GameFiddlesticks, valid_cards []int) []float64 {
	// probability the model will play this card
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

		}
		total_weight += weights[i]
	}
	return weights
}

func (m *FiddlesticksAiModelTheory2) calculateWeight(tricks_needed_factor float64, probability_of_winning float64) float64 {
	return 1 - math.Abs(tricks_needed_factor-probability_of_winning)
}
