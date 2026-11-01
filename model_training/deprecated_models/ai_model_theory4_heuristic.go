package deprecated

import (
	"fmt"
	"math"
	"os"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
)

/**
Theory model 4 keeps track of all cards based on both definitively known information
and inferences based on playstyle. For example, if another player bids high the
model will increase the likelihood that the player has high cards
*/

const t4LogitLimit = 5.0

type FiddlesticksAiModelTheory4 struct {
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
	bet_optimism               float64
	trick_chasing              float64
	trump_hoarding             float64
	void_hunting               float64
	vindictiveness             float64
	bid_trust                  float64
	bid_sigma                  float64
	levels                     []float64
	cards                      [52]*t4Location
	hand                       [52]*t4HandCard
	opponents                  []*t4Opponent // indexed by player id, nil for the model's own seat
	unknown_cards              int
	base_prob_kitty            float64
	base_prob_hand             float64
	locations_stale            bool
}

func createFiddlesticksAiModelTheory4(model_input map[string]string) *FiddlesticksAiModelTheory4 {
	m := &FiddlesticksAiModelTheory4{
		win_vs_all_trump_factor:    1.049,
		win_vs_all_nontrump_factor: 0.201,
		min_win_weight_factor:      0.813,
		bet_scale:                  0.906,
		aggressive_factor:          4.986,
		aggressive_factor_slope:    0.8,
		late_seat_bonus:            0.177,
		late_seat_bonus_slope:      0.9,
		trump_hold_factor:          0.016,
		void_exploit:               0.058,
		bet_optimism:               -0.171,
		trick_chasing:              0.166,
		trump_hoarding:             0.463,
		void_hunting:               0.043,
		vindictiveness:             0.255,
		bid_trust:                  0.541,
		bid_sigma:                  1,
	}
	m.initCards()
	targets := map[string]*float64{
		"win_vs_all_trump_factor":    &m.win_vs_all_trump_factor,
		"win_vs_all_nontrump_factor": &m.win_vs_all_nontrump_factor,
		"min_win_weight_factor":      &m.min_win_weight_factor,
		"bet_scale":                  &m.bet_scale,
		"aggressive_factor":          &m.aggressive_factor,
		"aggressive_factor_slope":    &m.aggressive_factor_slope,
		"late_seat_bonus":            &m.late_seat_bonus,
		"late_seat_bonus_slope":      &m.late_seat_bonus_slope,
		"trump_hold_factor":          &m.trump_hold_factor,
		"void_exploit":               &m.void_exploit,
		"bet_optimism":               &m.bet_optimism,
		"trick_chasing":              &m.trick_chasing,
		"trump_hoarding":             &m.trump_hoarding,
		"void_hunting":               &m.void_hunting,
		"vindictiveness":             &m.vindictiveness,
		"bid_trust":                  &m.bid_trust,
		"bid_sigma":                  &m.bid_sigma,
	}
	for k, v := range model_input {
		if target, ok := targets[k]; ok {
			*target = util.ParseFloat(v)
		} else {
			fmt.Fprintln(os.Stderr, "Unknown model input key for theory model 4:", k)
		}
	}
	return m
}

func (m *FiddlesticksAiModelTheory4) Bet(p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	min_p_suits := make(map[uint8]float64)
	card_max_ps := make(map[game_utils.StandardCardHash]float64)
	for _, card := range p.cards {
		lose_vs_one_p := float64(m.hand[t4Slot(card)].loseable_count) / float64(m.unknown_cards)
		win_vs_all_p := math.Pow(1-lose_vs_one_p, float64(len(f.players)-1))
		max_p := m.winVsAllP(win_vs_all_p, card.GetSuit() == f.trump.GetSuit())
		card_max_ps[card.Hash()] = max_p
		if min_p, ok := min_p_suits[card.GetSuit()]; !ok || min_p > max_p {
			min_p_suits[card.GetSuit()] = max_p
		}
	}
	bet := 0.0
	for _, card := range p.cards {
		bet += card_max_ps[card.Hash()]*m.min_win_weight_factor + min_p_suits[card.GetSuit()]*(1-m.min_win_weight_factor)
	}
	total_points := float64(f.round_points + f.trick_points)
	trick_ratio := 0.1
	if total_points > 0 {
		trick_ratio = float64(f.trick_points) / total_points
	}
	base_bet := util.Clamp(m.bet_scale*bet*(0.8+2.0*trick_ratio), 0, float64(f.round))
	return util.Clamp(base_bet+m.bet_optimism, 0, float64(f.round))
}

func (m *FiddlesticksAiModelTheory4) winVsAllP(base_p float64, trump bool) float64 {
	if trump {
		return util.Clamp(base_p*m.win_vs_all_trump_factor, 0, 1)
	}
	return util.Clamp(base_p*m.win_vs_all_nontrump_factor, 0, 1)
}

func (m *FiddlesticksAiModelTheory4) CardWeights(p *FiddlesticksPlayer, f *GameFiddlesticks, valid_cards []int) []float64 {
	if len(valid_cards) <= 1 {
		return []float64{1.0}
	}
	if m.locations_stale {
		m.updateLocationsWithBids(f)
		m.locations_stale = false
	}
	tricks_needed_factor := float64(int(p.bet)-int(p.tricks)) / float64(len(p.cards)-len(p.cards_played))
	if f.round_points == 0 && f.trick_points > 0 {
		tricks_needed_factor = 1.0 // maximize trick winning when contract bonus is 0
	}
	seat_fraction := float64(len(f.trick)) / float64(len(f.players)-1)
	aggressive := util.Clamp(t4Scaled(m.aggressive_factor, m.aggressive_factor_slope, len(f.players)), 0.2, 5)
	late_seat := util.Clamp(t4Scaled(m.late_seat_bonus, m.late_seat_bonus_slope, len(f.players)), 0, 2)
	current_winning_card, _ := f.winningTrickCard()
	weights := make([]float64, len(valid_cards))
	for i, card_index := range valid_cards {
		card := p.cards[card_index]
		prob_to_win := 0.0
		if f.cardBeatsCard(card, current_winning_card) {
			prob_to_win = m.winProbability(card, p, f)
		}
		weights[i] = t4Weight(aggressive*(1+late_seat*seat_fraction), tricks_needed_factor, prob_to_win) *
			m.heuristicFactor(card, f, tricks_needed_factor, prob_to_win) *
			m.behavioralFactor(card, f, tricks_needed_factor, prob_to_win)
		if m.vindictiveness != 0 {
			weights[i] *= math.Exp(util.Clamp(m.vindictiveness*m.opponentTrickEffect(card, p, f), -t4LogitLimit, t4LogitLimit))
		}
	}
	return weights
}

func (m *FiddlesticksAiModelTheory4) heuristicFactor(card *game_utils.StandardCard, f *GameFiddlesticks, tricks_needed_factor float64, prob_to_win float64) float64 {
	rank_fraction := float64(card.GetNumber()-2) / 12
	factor := 1.0
	if card.GetSuit() == f.trump.GetSuit() {
		factor *= 1 - m.trump_hold_factor*rank_fraction*(1-util.Clamp(tricks_needed_factor, 0, 1))
	}
	if len(f.trick) > 0 && card.GetSuit() != f.trick[0].GetSuit() && prob_to_win == 0 {
		factor *= 1 + m.void_exploit*(1-rank_fraction)
	}
	return factor
}

func (m *FiddlesticksAiModelTheory4) behavioralFactor(card *game_utils.StandardCard, f *GameFiddlesticks, tricks_needed_factor float64, prob_to_win float64) float64 {
	factor := 1.0
	rank_frac := float64(card.GetNumber()-2) / 12.0
	if m.trick_chasing != 0 {
		factor *= 1.0 + m.trick_chasing*tricks_needed_factor*prob_to_win
	}
	if m.trump_hoarding != 0 && card.GetSuit() == f.trump.GetSuit() {
		factor *= math.Max(0.1, 1.0-m.trump_hoarding*rank_frac*(1.0-util.Clamp(tricks_needed_factor, 0, 1)))
	}
	if m.void_hunting != 0 && len(f.trick) == 0 {
		for _, opp := range m.opponents {
			if opp != nil && opp.missing_suits[card.GetSuit()] {
				factor *= 1.0 + m.void_hunting
				break
			}
		}
	}
	return math.Max(0.001, factor)
}

func t4Scaled(base float64, slope float64, players int) float64 {
	return base + slope*(6/float64(players)-1)
}

func t4Sigmoid(aggressive float64, x float64) float64 {
	x = util.Clamp(x, 0.001, 2)
	return 2 / (1 + math.Pow(2/x-1, aggressive))
}

func t4Weight(aggressive float64, tricks_needed_factor float64, prob_to_win float64) float64 {
	return t4Sigmoid(aggressive, 1-math.Abs(t4Sigmoid(aggressive, tricks_needed_factor)-prob_to_win))
}

func t4ProbabilityToWin(card *game_utils.StandardCard, trump_suit uint8, num_players int) float64 {
	cards_that_can_beat_it := game_utils.StandardCardStatic.AceValue() - card.GetNumber()
	if card.GetSuit() != trump_suit {
		cards_that_can_beat_it += 12
	}
	return math.Pow(1-float64(cards_that_can_beat_it)/float64(50), float64(num_players))
}
