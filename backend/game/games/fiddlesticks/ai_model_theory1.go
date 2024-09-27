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
Theory model 1 is a model only accounting for each card individually
The hand as a whole and other players are not factored into this model
For playing cards, it doesn't account for future tricks, only how much it wants
	to win the current trick and whether a given card will win that trick
Calculations are based on how many other cards can beat this card
*/

type FiddlesticksAiModelTheory1 struct {
	// this factor determines the relationship between the weight factor and the
	// actual weight, where the weight factor is and initial estimate for the
	// weight given by = 1 - |tricks_needed_factor - probability_of_winning|
	// a value of 1 is linear, higher values are safer, lower values are more aggressive
	// it should be constrained to ~[1/5, 5]
	aggressive_factor float64
}

func createFiddlesticksAiModelTheory1(model_input map[string]string) *FiddlesticksAiModelTheory1 {
	m := &FiddlesticksAiModelTheory1{
		aggressive_factor: 0.8,
	}
	for k, v := range model_input {
		switch k {
		case "aggressive_factor":
			m.aggressive_factor = util.ParseFloat(v)
		default:
			fmt.Fprintln(os.Stderr, "Unknown model input key for theory model 1:", k)
		}
	}
	return m
}

func (m *FiddlesticksAiModelTheory1) ApplyUpdate(p *FiddlesticksPlayer, f *GameFiddlesticks, u *game.UpdateMessage) {
}

func (m *FiddlesticksAiModelTheory1) Bet(p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	bid := float64(0)
	trump_suit := f.trump.GetSuit()
	for _, card := range p.cards {
		bid += m.probabilityToWin(card, trump_suit, len(f.players)-1)
	}
	return bid
}

func (m FiddlesticksAiModelTheory1) CardWeights(p *FiddlesticksPlayer, f *GameFiddlesticks, valid_cards []int) []float64 {
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
		} else if len(f.trick)+1 >= len(f.players) {
			weights[i] = m.calculateWeight(tricks_needed_factor, 1)
		} else {
			probability_to_win := m.probabilityToWin(card, f.trump.GetSuit(), len(f.players)-1-len(f.trick))
			weights[i] = m.calculateWeight(tricks_needed_factor, probability_to_win)
		}
		total_weight += weights[i]
	}
	return weights
}

/** Probability that card will win */
func (m *FiddlesticksAiModelTheory1) probabilityToWin(card *game_utils.StandardCard, trump_suit uint8, num_players int) float64 {
	cards_that_can_beat_it := game_utils.StandardCardStatic.AceValue() - card.GetNumber()
	if card.GetSuit() != trump_suit {
		cards_that_can_beat_it += 12 // minus the card flipped up
	}
	// 50 because less the input card and less the trump card
	return math.Pow(1-float64(cards_that_can_beat_it)/float64(50), float64(num_players))
}

func (m *FiddlesticksAiModelTheory1) calculateWeight(tricks_needed_factor float64, probability_of_winning float64) float64 {
	weight_factor := 1 - math.Abs(m.sigmoid(tricks_needed_factor)-probability_of_winning)
	return m.sigmoid(weight_factor)
}

func (m *FiddlesticksAiModelTheory1) sigmoid(x float64) float64 {
	return 2 / (1 + math.Pow(2/x-1, m.aggressive_factor))
}
