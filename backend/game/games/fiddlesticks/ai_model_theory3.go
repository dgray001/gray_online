package fiddlesticks

import (
	"fmt"
	"math"
	"os"
	"strconv"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

/**
Theory model 3 keeps track of all cards based on definitely known information
Theory model 3 makes no inferences based on playstyle (e.g. another player's bid)
*/

type FiddlesticksAiModelTheory3 struct {
	// configurable model parameters
	win_vs_all_trump_factor    float64
	win_vs_all_nontrump_factor float64
	// internal data for a given game
	cards           map[uint16]*CardLocationProbability
	hand            map[uint16]map[uint16]bool // map of cards each card in hand can beat
	unknown_cards   int
	cards_in_kitty  int
	base_prob_kitty float64
	base_prob_hand  float64
}

func createFiddlesticksAiModelTheory3(model_input map[string]string) *FiddlesticksAiModelTheory3 {
	m := &FiddlesticksAiModelTheory3{
		// configurable model parameters
		win_vs_all_trump_factor:    1,
		win_vs_all_nontrump_factor: 1,
		// internal data for a given game
		cards:           make(map[uint16]*CardLocationProbability),
		unknown_cards:   0,
		base_prob_kitty: 0,
		base_prob_hand:  0,
	}
	for _, card := range game_utils.CreateStandardDeck().GetCards() {
		if card == nil {
			continue
		}
		probabilities := &CardLocationProbability{
			card:   card,
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

func (m *FiddlesticksAiModelTheory3) ApplyUpdate(p *FiddlesticksPlayer, f *GameFiddlesticks, u *game.UpdateMessage) {
	switch u.Kind {
	case "deal-round":
		m.hand = make(map[uint16]map[uint16]bool)
		for _, card := range p.cards {
			m.hand[card.Hash()] = make(map[uint16]bool)
		}
		m.unknown_cards = 52 - 1 - int(f.round)
		m.cards_in_kitty = 52 - 1 - len(f.players)*int(f.round)
		m.base_prob_kitty = float64(m.cards_in_kitty) / float64(m.unknown_cards)
		m.base_prob_hand = float64(f.round) / float64(m.unknown_cards)
		for hash, probabilities := range m.cards {
			probabilities.reset(f.players, p.player.Player_id)
			if hash == f.trump.Hash() {
				probabilities.isTrump()
			} else if util.MapContains(m.hand, hash) {
				probabilities.inHand()
			} else {
				probabilities.isUnknown(m.base_prob_kitty, m.base_prob_hand)
				for _, card := range p.cards {
					if f.cardBeatsCard(card, probabilities.card) {
						m.hand[card.Hash()][hash] = true
					}
				}
			}
		}
	case "bet":
		// Isn't a definitive marker of anything so nothing to do here
	case "play-card":
		suit := u.Content["card"].(gin.H)["suit"].(uint8)
		number := u.Content["card"].(gin.H)["number"].(uint8)
		hash := game_utils.StandardCardStatic.CreateCard(suit, number).Hash()
		m.cards[hash].cardPlayed()
		card, ok := m.hand[hash]
		if ok {
			delete(m.hand, hash)
		} else {
			delete(card, hash)
			m.unknown_cards--
		}
		m.base_prob_kitty = float64(m.cards_in_kitty) / float64(m.unknown_cards)
		for card_hash, probabilities := range m.cards {
			if hash == card_hash || !probabilities.unknown {
				continue
			}
			probabilities.kitty = m.base_prob_kitty
			for k := range probabilities.player {
				probabilities.player[k] = float64(len(f.players[k].cards)-len(f.players[k].cards_played)) / float64(m.unknown_cards)
			}
		}
		m.print()
	default:
		fmt.Fprintln(os.Stderr, "Unknown update kind for theory model 3:", u.Kind)
	}
}

func (m *FiddlesticksAiModelTheory3) Bet(p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	other_players := len(f.players) - 1
	min_p_suits := make(map[uint8]float64)
	for _, card := range p.cards {
		lose_vs_one_p := float64(len(m.hand[card.Hash()])) / float64(m.unknown_cards)
		win_vs_all_p := math.Pow(1-lose_vs_one_p, float64(other_players))
		max_p := m.winVsAllP(win_vs_all_p, card.GetSuit() == f.trump.GetSuit())
		min_p, ok := min_p_suits[card.GetSuit()]
		if !ok || min_p > max_p {
			min_p_suits[card.GetSuit()] = max_p
		}
	}
	return 0
}

// accounts for leading card which can help increase chance of winning since others follow suit
func (m *FiddlesticksAiModelTheory3) winVsAllP(base_p float64, trump bool) float64 {
	if trump {
		return base_p * m.win_vs_all_trump_factor
	}
	return base_p * m.win_vs_all_nontrump_factor
}

func (m *FiddlesticksAiModelTheory3) CardWeights(p *FiddlesticksPlayer, f *GameFiddlesticks, valid_cards []int) []float64 {
	weights := make([]float64, len(valid_cards))
	for i := range weights {
		weights[i] = 1
	}
	return weights
}

func (m *FiddlesticksAiModelTheory3) print() {
	for _, card := range m.cards {
		fmt.Println(card.printString())
	}
	for _, card := range m.hand {
		fmt.Println(strconv.FormatFloat(float64(len(card))/float64(m.unknown_cards), 'f', 3, 64))
	}
	fmt.Println("Unknown cards:", m.unknown_cards)
	fmt.Println("Prob kitty:", strconv.FormatFloat(m.base_prob_kitty, 'f', 3, 64))
	fmt.Println("Prob hand:", strconv.FormatFloat(m.base_prob_hand, 'f', 3, 64))
}
