package fiddlesticks

import (
	"fmt"
	"math"
	"os"

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
	bet_scale                  float64
	aggressive_factor          float64
	aggressive_factor_slope    float64
	late_seat_bonus            float64
	late_seat_bonus_slope      float64
	trump_hold_factor          float64
	void_exploit               float64
	// internal data for a given game
	// object describing where a card might be
	cards [52]*CardLocationProbability
	// objects describing relevant informatino about each card in the hand
	hand [52]*HandCardDetails
	// objects describing relevant information about opponents
	created_opponents bool
	opponents         []*PlayerDetails // indexed by player id, nil for the model's own seat
	// num of cards that are unknown
	unknown_cards int
	// num of cards in the kitty
	cards_in_kitty int
	// base prob that a card is in the kitty (cards_in_kitty / unknown_cards)
	base_prob_kitty float64
	// base prob that a card is in a different player's hand
	base_prob_hand float64
	// whether plays have been seen since the card locations were last fitted
	locations_stale bool
}

func scaledByPlayers(base float64, slope float64, players int) float64 {
	return base + slope*(6/float64(players)-1)
}

func cardSlot(card *game_utils.StandardCard) int {
	return int(card.GetSuit()-1)*13 + int(card.GetNumber()-2)
}

func createFiddlesticksAiModelTheory3(model_input map[string]string) *FiddlesticksAiModelTheory3 {
	m := &FiddlesticksAiModelTheory3{
		// configurable model parameters
		win_vs_all_trump_factor:    1.136,
		win_vs_all_nontrump_factor: 0.203,
		min_win_weight_factor:      0.8,
		bet_scale:                  0.884,
		aggressive_factor:          1.09,
		aggressive_factor_slope:    0.8,
		late_seat_bonus:            0.06,
		late_seat_bonus_slope:      0.9,
		trump_hold_factor:          0.396,
		void_exploit:               0.194,
		// internal data for a given game
		created_opponents: false,
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
		}
		m.cards[cardSlot(card)] = probabilities
	}
	for k, v := range model_input {
		switch k {
		case "win_vs_all_trump_factor":
			m.win_vs_all_trump_factor = util.ParseFloat(v)
		case "win_vs_all_nontrump_factor":
			m.win_vs_all_nontrump_factor = util.ParseFloat(v)
		case "min_win_weight_factor":
			m.min_win_weight_factor = util.ParseFloat(v)
		case "bet_scale":
			m.bet_scale = util.ParseFloat(v)
		case "aggressive_factor":
			m.aggressive_factor = util.ParseFloat(v)
		case "aggressive_factor_slope":
			m.aggressive_factor_slope = util.ParseFloat(v)
		case "late_seat_bonus":
			m.late_seat_bonus = util.ParseFloat(v)
		case "late_seat_bonus_slope":
			m.late_seat_bonus_slope = util.ParseFloat(v)
		case "trump_hold_factor":
			m.trump_hold_factor = util.ParseFloat(v)
		case "void_exploit":
			m.void_exploit = util.ParseFloat(v)
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
			m.opponents = make([]*PlayerDetails, len(f.players))
			for i := range f.players {
				if i == p.player.Player_id {
					continue
				}
				m.opponents[i] = createPlayerDetails()
			}
			m.created_opponents = true
		}
		for _, player := range m.opponents {
			if player != nil {
				player.dealHand()
			}
		}
		m.hand = [52]*HandCardDetails{}
		for _, card := range p.cards {
			m.hand[cardSlot(card)] = createHandCardDetails(card)
		}
		m.unknown_cards = 52 - 1 - int(f.round)
		m.cards_in_kitty = 52 - 1 - len(f.players)*int(f.round)
		m.base_prob_kitty = float64(m.cards_in_kitty) / float64(m.unknown_cards)
		m.base_prob_hand = float64(f.round) / float64(m.unknown_cards)
		m.locations_stale = false
		for _, probabilities := range m.cards {
			slot := cardSlot(probabilities.card)
			probabilities.reset(f.players, p.player.Player_id)
			if slot == cardSlot(f.trump) {
				probabilities.isTrump()
			} else if m.hand[slot] != nil {
				probabilities.inHand()
			} else {
				probabilities.isUnknown(m.base_prob_kitty, m.base_prob_hand)
				for _, card := range p.cards {
					if f.cardBeatsCard(card, probabilities.card) {
						m.hand[cardSlot(card)].addWinCard(slot)
					} else if f.cardBeatsCard(probabilities.card, card) {
						m.hand[cardSlot(card)].addLoseCard(slot)
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
		played_card := game_utils.StandardCardStatic.CreateCard(suit, number)
		slot := cardSlot(played_card)
		m.cards[slot].cardPlayed()
		if m.hand[slot] != nil {
			m.hand[slot] = nil
		} else {
			for _, details := range m.hand {
				if details != nil {
					details.cardPlayed(slot)
				}
			}
			m.unknown_cards--
		}
		if opponent := m.opponents[player_id]; opponent != nil && suit != lead_suit {
			opponent.missingSuit(lead_suit)
		}
		m.locations_stale = true
	default:
		fmt.Fprintln(os.Stderr, "Unknown update kind for theory model 3:", u.Kind)
	}
}

const LOCATION_FITTING_ROUNDS = 20
const LOCATION_FITTING_TOLERANCE = 1e-6

type LocationTarget struct {
	column int
	amount float64
}

// counts[i] is how many cards pool[i] stands for
func fitLocations(pool []*CardLocationProbability, counts []float64, targets []LocationTarget) {
	previous := make([]float64, 0, len(pool)*(len(targets)+1))
	for range LOCATION_FITTING_ROUNDS {
		for _, location := range pool {
			location.normalize()
		}
		if settled(pool, previous) {
			break
		}
		previous = previous[:0]
		for _, location := range pool {
			previous = append(previous, location.kitty)
			previous = append(previous, location.player...)
		}
		for _, target := range targets {
			scaleToTarget(pool, counts, target.column, target.amount)
		}
	}
	for _, location := range pool {
		location.normalize()
	}
}

// whether no probability moved more than the tolerance since the previous round
func settled(pool []*CardLocationProbability, previous []float64) bool {
	if len(previous) == 0 {
		return false
	}
	i := 0
	for _, location := range pool {
		if math.Abs(location.kitty-previous[i]) > LOCATION_FITTING_TOLERANCE {
			return false
		}
		i++
		for _, value := range location.player {
			if math.Abs(value-previous[i]) > LOCATION_FITTING_TOLERANCE {
				return false
			}
			i++
		}
	}
	return true
}

func scaleToTarget(pool []*CardLocationProbability, counts []float64, id int, target float64) {
	column := float64(0)
	for i, location := range pool {
		column += counts[i] * location.column(id)
	}
	if column == 0 {
		return
	}
	for _, location := range pool {
		location.scaleColumn(id, target/column)
	}
}

// cards of one suit are interchangeable, so one fitted card per suit is copied to the rest
func fitLocationsBySuit(pool []*CardLocationProbability, targets []LocationTarget) {
	var suit_cards [5][]*CardLocationProbability
	for _, location := range pool {
		suit_cards[location.card.GetSuit()] = append(suit_cards[location.card.GetSuit()], location)
	}
	representatives := make([]*CardLocationProbability, 0, 4)
	counts := make([]float64, 0, 4)
	for _, cards := range suit_cards {
		if len(cards) > 0 {
			representatives = append(representatives, cards[0])
			counts = append(counts, float64(len(cards)))
		}
	}
	fitLocations(representatives, counts, targets)
	for _, cards := range suit_cards {
		for i := 1; i < len(cards); i++ {
			cards[i].copyLocations(cards[0])
		}
	}
}

func setUniformLocations(pool []*CardLocationProbability, targets []LocationTarget) {
	for _, location := range pool {
		for _, target := range targets {
			location.setColumn(target.column, target.amount/float64(len(pool)))
		}
	}
}

func (m *FiddlesticksAiModelTheory3) updateLocations(f *GameFiddlesticks) {
	pool := make([]*CardLocationProbability, 0, m.unknown_cards)
	for _, location := range m.cards {
		if location.unknown {
			location.isUnknown(1, 1)
			pool = append(pool, location)
		}
	}
	targets := []LocationTarget{{KITTY_COLUMN, float64(len(pool))}}
	voided := false
	for id, opponent := range m.opponents {
		if opponent == nil {
			continue
		}
		hand_size := float64(len(f.players[id].cards) - len(f.players[id].cards_played))
		targets = append(targets, LocationTarget{id, hand_size})
		targets[0].amount -= hand_size
		for _, location := range pool {
			if opponent.missing_suits[location.card.GetSuit()] {
				location.player[id] = 0
				voided = true
			}
		}
	}
	if !voided {
		setUniformLocations(pool, targets)
		return
	}
	fitLocationsBySuit(pool, targets)
}

func (m *FiddlesticksAiModelTheory3) Bet(p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	other_players := len(f.players) - 1
	min_p_suits := make(map[uint8]float64)
	card_max_ps := make(map[game_utils.StandardCardHash]float64)
	for _, card := range p.cards {
		lose_vs_one_p := float64(m.hand[cardSlot(card)].loseable_count) / float64(m.unknown_cards)
		win_vs_all_p := math.Pow(1-lose_vs_one_p, float64(other_players))
		max_p := m.winVsAllP(win_vs_all_p, card.GetSuit() == f.trump.GetSuit())
		card_max_ps[card.Hash()] = max_p
		min_p, ok := min_p_suits[card.GetSuit()]
		if !ok || min_p > max_p {
			min_p_suits[card.GetSuit()] = max_p
		}
	}
	bet := float64(0)
	for _, card := range p.cards {
		bet += m.minWinP(card_max_ps[card.Hash()], min_p_suits[card.GetSuit()])
	}
	return util.Clamp(m.bet_scale*bet, 0, float64(f.round))
}

// accounts for leading card which can help increase chance of winning since others follow suit
// or if a low trump card accounts for not leading card
func (m *FiddlesticksAiModelTheory3) winVsAllP(base_p float64, trump bool) float64 {
	if trump {
		return util.Clamp(base_p*m.win_vs_all_trump_factor, 0, 1)
	}
	return util.Clamp(base_p*m.win_vs_all_nontrump_factor, 0, 1)
}

// accounts for being able to play a lower card of the same suit
func (m *FiddlesticksAiModelTheory3) minWinP(max_p float64, suit_min_p float64) float64 {
	return max_p*m.min_win_weight_factor + suit_min_p*(1-m.min_win_weight_factor)
}

func (m *FiddlesticksAiModelTheory3) CardWeights(p *FiddlesticksPlayer, f *GameFiddlesticks, valid_cards []int) []float64 {
	if len(valid_cards) == 1 {
		return []float64{1}
	}
	if m.locations_stale {
		m.updateLocations(f)
		m.locations_stale = false
	}
	weights := make([]float64, len(valid_cards))
	tricks_needed := int(p.bet) - int(p.tricks)
	tricks_left := len(p.cards) - len(p.cards_played)
	tricks_needed_factor := float64(tricks_needed) / float64(tricks_left)
	seat_fraction := float64(len(f.trick)) / float64(len(f.players)-1)
	aggressive_factor := util.Clamp(scaledByPlayers(m.aggressive_factor, m.aggressive_factor_slope, len(f.players)), 0.2, 5)
	late_seat_bonus := util.Clamp(scaledByPlayers(m.late_seat_bonus, m.late_seat_bonus_slope, len(f.players)), 0, 2)
	weighting := &FiddlesticksAiModelTheory1{aggressive_factor: aggressive_factor * (1 + late_seat_bonus*seat_fraction)}
	current_winning_card, _ := f.winningTrickCard()
	for i, card_index := range valid_cards {
		card := p.cards[card_index]
		probability_to_win := float64(0)
		if f.cardBeatsCard(card, current_winning_card) {
			probability_to_win = m.winProbability(card, p, f)
		}
		weights[i] = weighting.calculateWeight(tricks_needed_factor, probability_to_win) *
			m.heuristicFactor(card, f, tricks_needed_factor, probability_to_win)
	}
	return weights
}

func (m *FiddlesticksAiModelTheory3) heuristicFactor(card *game_utils.StandardCard, f *GameFiddlesticks, tricks_needed_factor float64, probability_to_win float64) float64 {
	rank_fraction := float64(card.GetNumber()-2) / 12
	factor := float64(1)
	if card.GetSuit() == f.trump.GetSuit() {
		factor *= 1 - m.trump_hold_factor*rank_fraction*(1-util.Clamp(tricks_needed_factor, 0, 1))
	}
	if len(f.trick) > 0 && card.GetSuit() != f.trick[0].GetSuit() && probability_to_win == 0 {
		factor *= 1 + m.void_exploit*(1-rank_fraction)
	}
	return factor
}

func (m *FiddlesticksAiModelTheory3) winProbability(card *game_utils.StandardCard, p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	lead_suit := card.GetSuit()
	if len(f.trick) > 0 {
		lead_suit = f.trick[0].GetSuit()
	}
	probability := float64(1)
	for i := 1; i < len(f.players)-len(f.trick); i++ {
		opponent_id := (p.player.Player_id + i) % len(f.players)
		probability *= 1 - m.beatProbability(card, opponent_id, lead_suit, f)
	}
	return probability
}

func (m *FiddlesticksAiModelTheory3) beatProbability(card *game_utils.StandardCard, opponent_id int, lead_suit uint8, f *GameFiddlesticks) float64 {
	var lead_total, lead_beating, other_total, other_beating float64
	no_lead := float64(1)
	for _, location := range m.cards {
		weight := location.player[opponent_id]
		beats := f.cardBeatsCard(location.card, card)
		if location.card.GetSuit() != lead_suit {
			other_total += weight
			if beats {
				other_beating += weight
			}
			continue
		}
		lead_total += weight
		no_lead *= 1 - weight
		if beats {
			lead_beating += weight
		}
	}
	beat_probability := float64(0)
	if lead_total > 0 {
		beat_probability += (1 - no_lead) * lead_beating / lead_total
	}
	if other_total > 0 {
		beat_probability += no_lead * other_beating / other_total
	}
	return beat_probability
}
