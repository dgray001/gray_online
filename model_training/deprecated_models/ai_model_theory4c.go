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
Theory model 4c samples complete deals consistent with what it knows, weights them by
how well each opponent's bid fits their sampled hand, and plays each candidate card out
to the end of the round in every sampled world to choose by expected contract utility
*/

const t4cLogitLimit = 5.0

type sampledWorld struct {
	hands  [][]*game_utils.StandardCard
	weight float64
}

type FiddlesticksAiModelTheory4c struct {
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
	worlds                     float64
	bid_sigma                  float64
	paranoia                   float64
	bid_trust                  float64
	levels                     []float64
	cards                      [52]*t4cLocation
	hand                       [52]*t4cHandCard
	opponents                  []*t4cOpponent // indexed by player id, nil for the model's own seat
	unknown_cards              int
	base_prob_kitty            float64
	base_prob_hand             float64
	locations_stale            bool
}

func createFiddlesticksAiModelTheory4c(model_input map[string]string) *FiddlesticksAiModelTheory4c {
	m := &FiddlesticksAiModelTheory4c{
		win_vs_all_trump_factor:    1.046,
		win_vs_all_nontrump_factor: 0.200,
		min_win_weight_factor:      0.869,
		bet_scale:                  0.965,
		bet_optimism:               -0.079,
		temperature:                0.054,
		overtrick_aversion:         0.698,
		undertrick_aversion:        0.474,
		worlds:                     16,
		bid_sigma:                  1.0,
		paranoia:                   0.144,
		bid_trust:                  0.503,
		vindictiveness:             1.654,
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
		"worlds":                     &m.worlds,
		"bid_sigma":                  &m.bid_sigma,
		"paranoia":                   &m.paranoia,
		"bid_trust":                  &m.bid_trust,
	}
	for k, v := range model_input {
		if target, ok := targets[k]; ok {
			*target = util.ParseFloat(v)
		} else {
			fmt.Fprintln(os.Stderr, "Unknown model input key for theory model 4c:", k)
		}
	}
	return m
}

func (m *FiddlesticksAiModelTheory4c) Bet(p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	min_p_suits := make(map[uint8]float64)
	card_max_ps := make(map[game_utils.StandardCardHash]float64)
	for _, card := range p.cards {
		lose_vs_one_p := float64(m.hand[t4cSlot(card)].loseable_count) / float64(m.unknown_cards)
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

func (m *FiddlesticksAiModelTheory4c) winVsAllP(base_p float64, trump bool) float64 {
	if trump {
		return util.Clamp(base_p*m.win_vs_all_trump_factor, 0, 1)
	}
	return util.Clamp(base_p*m.win_vs_all_nontrump_factor, 0, 1)
}

func (m *FiddlesticksAiModelTheory4c) CardWeights(p *FiddlesticksPlayer, f *GameFiddlesticks, valid_cards []int) []float64 {
	if len(valid_cards) <= 1 {
		return []float64{1.0}
	}
	if m.locations_stale {
		m.updateLocations(f)
		m.locations_stale = false
	}
	worlds := m.sampleWorlds(p, f)
	utilities := make([]float64, len(valid_cards))
	shifts := make([]float64, len(valid_cards))
	for i, card_index := range valid_cards {
		utilities[i] = m.evaluateInWorlds(card_index, worlds, p, f)
		shifts[i] = m.heuristicAdjustment(p.cards[card_index], p, f)
	}
	return t4cSoftmax(utilities, shifts, m.temperature)
}

func (m *FiddlesticksAiModelTheory4c) contractUtility(gap int, bet int, f *GameFiddlesticks) float64 {
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

func t4cSoftmax(utilities []float64, shifts []float64, temp float64) []float64 {
	t := math.Max(0.01, temp)
	logits := make([]float64, len(utilities))
	for i, u := range utilities {
		logits[i] = u/t + shifts[i]
	}
	max_logit := slices.Max(logits)
	weights := make([]float64, len(logits))
	for i, l := range logits {
		weights[i] = math.Exp(util.Clamp(l-max_logit, -t4cLogitLimit, 0.0))
	}
	return weights
}

func (m *FiddlesticksAiModelTheory4c) evaluateInWorlds(card_index int, worlds []*sampledWorld, p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	total, weighted, worst := 0.0, 0.0, math.Inf(1)
	for _, w := range worlds {
		if w.weight == 0 {
			continue
		}
		utility := m.contractUtility(m.rolloutGap(card_index, w, p, f), int(p.bet), f)
		total += w.weight
		weighted += w.weight * utility
		worst = math.Min(worst, utility)
	}
	if total == 0 {
		return 0.0
	}
	return (1.0-m.paranoia)*weighted/total + m.paranoia*worst
}
