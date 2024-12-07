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
	min_win_weight_factor      float64
	// internal data for a given game
	// object describing where a card might be
	cards map[game_utils.StandardCardHash]*CardLocationProbability
	// objects describing relevant informatino about each card in the hand
	hand map[game_utils.StandardCardHash]*HandCardDetails
	// objects describing relevant information about opponents
	created_opponents bool
	opponents         map[int]*PlayerDetails
	// num of cards that are unknown
	unknown_cards int
	// num of cards in the kitty
	cards_in_kitty int
	// base prob that a card is in the kitty (cards_in_kitty / unknown_cards)
	base_prob_kitty float64
	// base prob that a card is in a different player's hand
	base_prob_hand float64
}

func createFiddlesticksAiModelTheory3(model_input map[string]string) *FiddlesticksAiModelTheory3 {
	m := &FiddlesticksAiModelTheory3{
		// configurable model parameters
		win_vs_all_trump_factor:    1,
		win_vs_all_nontrump_factor: 1,
		min_win_weight_factor:      0.5,
		// internal data for a given game
		cards:             make(map[game_utils.StandardCardHash]*CardLocationProbability),
		created_opponents: false,
		opponents:         make(map[int]*PlayerDetails),
		unknown_cards:     0,
		base_prob_kitty:   0,
		base_prob_hand:    0,
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
	for k, v := range model_input {
		switch k {
		case "win_vs_all_trump_factor":
			m.win_vs_all_trump_factor = util.ParseFloat(v)
		case "win_vs_all_nontrump_factor":
			m.win_vs_all_nontrump_factor = util.ParseFloat(v)
		case "min_win_weight_factor":
			m.min_win_weight_factor = util.ParseFloat(v)
		default:
			fmt.Fprintln(os.Stderr, "Unknown model input key for theory model 3:", k)
		}
	}
	return m
}

func (m *FiddlesticksAiModelTheory3) ApplyUpdate(p *FiddlesticksPlayer, f *GameFiddlesticks, u *game.UpdateMessage) {
	switch u.Kind {
	case "deal-round":
		if !m.created_opponents {
			for i := range f.players {
				if i == p.player.Player_id {
					continue
				}
				m.opponents[i] = createPlayerDetails()
			}
		}
		for _, player := range m.opponents {
			player.dealHand()
		}
		m.hand = make(map[game_utils.StandardCardHash]*HandCardDetails)
		for _, card := range p.cards {
			m.hand[card.Hash()] = createHandCardDetails(card)
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
						m.hand[card.Hash()].addWinCard(hash)
					} else if f.cardBeatsCard(probabilities.card, card) {
						m.hand[card.Hash()].addLoseCard(hash)
					}
				}
			}
		}
	case "bet":
		// Isn't a definitive marker of anything so nothing to do here
	case "play-card":
		suit := u.Content["card"].(gin.H)["suit"].(uint8)
		lead_suit := u.Content["lead_suit"].(uint8)
		number := u.Content["card"].(gin.H)["number"].(uint8)
		player_id := u.Content["player_id"].(int)
		hash := game_utils.StandardCardStatic.CreateCard(suit, number).Hash()
		m.cards[hash].cardPlayed()
		_, ok := m.hand[hash]
		if ok {
			delete(m.hand, hash)
		} else {
			for _, details := range m.hand {
				details.cardPlayed(hash)
			}
			m.unknown_cards--
			m.base_prob_kitty = float64(m.cards_in_kitty) / float64(m.unknown_cards)
		}
		for card_hash, probabilities := range m.cards {
			if hash == card_hash || !probabilities.unknown {
				continue
			}
			probabilities.kitty = m.base_prob_kitty
			for k := range probabilities.player {
				probabilities.player[k] = float64(len(f.players[k].cards)-len(f.players[k].cards_played)) / float64(m.unknown_cards)
			}
		}
		if suit != lead_suit {
			m.opponents[player_id].missingSuit(lead_suit)
			// account for missing suit in card probabilities
		}
	default:
		fmt.Fprintln(os.Stderr, "Unknown update kind for theory model 3:", u.Kind)
	}
}

func (m *FiddlesticksAiModelTheory3) Bet(p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	other_players := len(f.players) - 1
	min_p_suits := make(map[uint8]float64)
	card_max_ps := make(map[game_utils.StandardCardHash]float64)
	max_p_total := float64(0)
	min_p_total := float64(0)
	for _, card := range p.cards {
		lose_vs_one_p := float64(len(m.hand[card.Hash()].beatable_cards)) / float64(m.unknown_cards)
		win_vs_all_p := math.Pow(1-lose_vs_one_p, float64(other_players))
		max_p := m.winVsAllP(win_vs_all_p, card.GetSuit() == f.trump.GetSuit())
		card_max_ps[card.Hash()] = max_p
		max_p_total += max_p_total
		min_p, ok := min_p_suits[card.GetSuit()]
		if !ok || min_p > max_p {
			min_p_suits[card.GetSuit()] = max_p
		}
	}
	for _, card := range p.cards {
		min_p_total += m.minWinP(card_max_ps[card.Hash()], min_p_suits[card.GetSuit()])
	}
	return 0
}

// accounts for leading card which can help increase chance of winning since others follow suit
// or if a low trump card accounts for not leading card
func (m *FiddlesticksAiModelTheory3) winVsAllP(base_p float64, trump bool) float64 {
	if trump {
		return base_p * m.win_vs_all_trump_factor
	}
	return base_p * m.win_vs_all_nontrump_factor
}

// accounts for being able to play a lower card of the same suit
func (m *FiddlesticksAiModelTheory3) minWinP(max_p float64, suit_min_p float64) float64 {
	return max_p*m.min_win_weight_factor + suit_min_p*(1-m.min_win_weight_factor)
}

func (m *FiddlesticksAiModelTheory3) CardWeights(p *FiddlesticksPlayer, f *GameFiddlesticks, valid_cards []int) []float64 {
	weights := make([]float64, len(valid_cards))
	current_winning_card, _ := f.winningTrickCard()
	//ticks_needed := p.bet - p.tricks
	//tricks_left := len(p.cards) - len(p.cards_played)
	//tricks_needed_factor := float64(tricks_needed) / float64(tricks_left)
	for _, card_index := range valid_cards {
		card := p.cards[card_index]
		if !f.cardBeatsCard(card, current_winning_card) {
		} else if len(f.trick) == len(f.players)-1 {
		} else {
		}
	}
	return weights
}

func (m *FiddlesticksAiModelTheory3) print() {
	for _, card := range m.cards {
		fmt.Println(card.printString())
	}
	for _, card := range m.hand {
		fmt.Println(strconv.FormatFloat(float64(len(card.beatable_cards))/float64(m.unknown_cards), 'f', 3, 64))
	}
	fmt.Println("Unknown cards:", m.unknown_cards)
	fmt.Println("Prob kitty:", strconv.FormatFloat(m.base_prob_kitty, 'f', 3, 64))
	fmt.Println("Prob hand:", strconv.FormatFloat(m.base_prob_hand, 'f', 3, 64))
}
