package deprecated

import (
	"fmt"
	"math"
	"os"
	"slices"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
)

/**
Theory model 4a plays by a contract planner: it estimates each remaining card's
chance to win a trick and picks cards to maximize expected contract utility,
using only definitively known information (no bid inference)
*/

const t4aLogitLimit = 5.0

type FiddlesticksAiModelTheory4a struct {
	win_vs_all_trump_factor    float64
	win_vs_all_nontrump_factor float64
	min_win_weight_factor      float64
	bet_scale                  float64
	bet_optimism               float64
	trump_hoarding             float64
	void_hunting               float64
	vindictiveness             float64
	temperature                float64
	overtrick_aversion         float64
	undertrick_aversion        float64
	levels                     []float64
	cards                      [52]*t4aLocation
	hand                       [52]*t4aHandCard
	opponents                  []*t4aOpponent // indexed by player id, nil for the model's own seat
	unknown_cards              int
	base_prob_kitty            float64
	base_prob_hand             float64
	locations_stale            bool
}

func createFiddlesticksAiModelTheory4a(model_input map[string]string) *FiddlesticksAiModelTheory4a {
	m := &FiddlesticksAiModelTheory4a{
		win_vs_all_trump_factor:    1.156,
		win_vs_all_nontrump_factor: 0.445,
		min_win_weight_factor:      0.510,
		bet_scale:                  1.138,
		bet_optimism:               -0.348,
		temperature:                0.051,
		overtrick_aversion:         1.203,
		undertrick_aversion:        2.785,
		trump_hoarding:             0.415,
		void_hunting:               0.332,
		vindictiveness:             1.169,
	}
	m.initCards()
	targets := map[string]*float64{
		"win_vs_all_trump_factor":    &m.win_vs_all_trump_factor,
		"win_vs_all_nontrump_factor": &m.win_vs_all_nontrump_factor,
		"min_win_weight_factor":      &m.min_win_weight_factor,
		"bet_scale":                  &m.bet_scale,
		"bet_optimism":               &m.bet_optimism,
		"trump_hoarding":             &m.trump_hoarding,
		"void_hunting":               &m.void_hunting,
		"vindictiveness":             &m.vindictiveness,
		"planner_temperature":        &m.temperature,
		"overtrick_aversion":         &m.overtrick_aversion,
		"undertrick_aversion":        &m.undertrick_aversion,
	}
	for k, v := range model_input {
		if target, ok := targets[k]; ok {
			*target = util.ParseFloat(v)
		} else {
			fmt.Fprintln(os.Stderr, "Unknown model input key for theory model 4a:", k)
		}
	}
	return m
}

func (m *FiddlesticksAiModelTheory4a) Bet(p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	min_p_suits := make(map[uint8]float64)
	card_max_ps := make(map[game_utils.StandardCardHash]float64)
	for _, card := range p.cards {
		lose_vs_one_p := float64(m.hand[t4aSlot(card)].loseable_count) / float64(m.unknown_cards)
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
	base_bet := util.Clamp(m.bet_scale*bet, 0, float64(f.round))
	return util.Clamp(base_bet+m.bet_optimism, 0, float64(f.round))
}

func (m *FiddlesticksAiModelTheory4a) winVsAllP(base_p float64, trump bool) float64 {
	if trump {
		return util.Clamp(base_p*m.win_vs_all_trump_factor, 0, 1)
	}
	return util.Clamp(base_p*m.win_vs_all_nontrump_factor, 0, 1)
}

func (m *FiddlesticksAiModelTheory4a) CardWeights(p *FiddlesticksPlayer, f *GameFiddlesticks, valid_cards []int) []float64 {
	if len(valid_cards) <= 1 {
		return []float64{1.0}
	}
	if m.locations_stale {
		m.updateLocations(f)
		m.locations_stale = false
	}
	utilities := make([]float64, len(valid_cards))
	shifts := make([]float64, len(valid_cards))
	needed := int(p.bet) - int(p.tricks)
	for i, card_index := range valid_cards {
		card := p.cards[card_index]
		utilities[i] = m.evaluateCardQ(m.currentWinProbability(card, p, f), needed, m.otherWinProbs(p, t4aSlot(card), f), int(p.bet), f)
		shifts[i] = m.heuristicAdjustment(card, p, f)
	}
	return t4aSoftmax(utilities, shifts, m.temperature)
}

func (m *FiddlesticksAiModelTheory4a) contractUtility(gap int, bet int, f *GameFiddlesticks) float64 {
	payoff := float64(f.round_points + f.trick_points*uint16(bet))
	if payoff <= 0 {
		payoff = 1.0
	}
	switch {
	case gap == 0:
		return payoff / 10.0
	case gap > 0:
		return -m.undertrick_aversion * float64(gap) * (payoff / 10.0)
	default:
		return m.overtrick_aversion * float64(gap) * (payoff / 10.0)
	}
}

func t4aPmf(probabilities []float64) []float64 {
	dp := []float64{1.0}
	for _, prob := range probabilities {
		next := make([]float64, len(dp)+1)
		for k, v := range dp {
			next[k] += v * (1.0 - prob)
			next[k+1] += v * prob
		}
		dp = next
	}
	return dp
}

func (m *FiddlesticksAiModelTheory4a) expectedFutureUtility(win_probs []float64, needed int, bet int, f *GameFiddlesticks) float64 {
	expected := 0.0
	for future_wins, prob := range t4aPmf(win_probs) {
		expected += prob * m.contractUtility(needed-future_wins, bet, f)
	}
	return expected
}

func (m *FiddlesticksAiModelTheory4a) evaluateCardQ(win_prob float64, needed int, other_probs []float64, bet int, f *GameFiddlesticks) float64 {
	return win_prob*m.expectedFutureUtility(other_probs, needed-1, bet, f) + (1.0-win_prob)*m.expectedFutureUtility(other_probs, needed, bet, f)
}

func t4aSoftmax(utilities []float64, shifts []float64, temp float64) []float64 {
	t := math.Max(0.01, temp)
	logits := make([]float64, len(utilities))
	for i, u := range utilities {
		logits[i] = u/t + shifts[i]
	}
	max_logit := slices.Max(logits)
	weights := make([]float64, len(logits))
	for i, l := range logits {
		weights[i] = math.Exp(util.Clamp(l-max_logit, -t4aLogitLimit, 0.0))
	}
	return weights
}
