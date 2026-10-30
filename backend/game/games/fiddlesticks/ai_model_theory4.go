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
Theory model 4 keeps track of all cards based on both definitively known information
and inferences based on playstyle. For example, if another player bids high the
model will increase the likelihood that the player has high cards
*/

type FiddlesticksAiModelTheory4 struct {
	FiddlesticksAiModelTheory3
	bet_optimism   float64
	trick_chasing  float64
	trump_hoarding float64
	void_hunting   float64
	vindictiveness float64
	bid_trust      float64
	bid_sigma      float64
	levels         []float64
}

func createFiddlesticksAiModelTheory4(model_input map[string]string) *FiddlesticksAiModelTheory4 {
	m := &FiddlesticksAiModelTheory4{
		FiddlesticksAiModelTheory3: *createFiddlesticksAiModelTheory3(model_input),
		bid_sigma:                  1,
	}
	for k, v := range model_input {
		switch k {
		case "bid_sigma":
			m.bid_sigma = util.ParseFloat(v)
		case "bet_optimism":
			m.bet_optimism = util.ParseFloat(v)
		case "trick_chasing":
			m.trick_chasing = util.ParseFloat(v)
		case "trump_hoarding":
			m.trump_hoarding = util.ParseFloat(v)
		case "void_hunting":
			m.void_hunting = util.ParseFloat(v)
		case "vindictiveness":
			m.vindictiveness = util.ParseFloat(v)
		case "bid_trust":
			m.bid_trust = util.ParseFloat(v)
		default:
			fmt.Fprintln(os.Stderr, "Unknown model input key for theory model 4:", k)
		}
	}
	return m
}

func (m *FiddlesticksAiModelTheory4) ApplyUpdate(p *FiddlesticksPlayer, f *GameFiddlesticks, u *game.UpdateMessage) {
	m.FiddlesticksAiModelTheory3.ApplyUpdate(p, f, u)
	switch u.Kind {
	case "deal-round":
		if len(m.levels) != len(f.players) {
			m.levels = make([]float64, len(f.players))
			for i := range m.levels {
				m.levels[i] = 1.0
			}
		}
	}
}

func (m *FiddlesticksAiModelTheory4) updateLocationsWithBids(f *GameFiddlesticks) {
	m.updateLocations(f)
	pool := m.unknownPool()
	if m.bid_trust == 0 || len(pool) == 0 {
		return
	}
	m.weightByBids(pool, f)
	counts := make([]float64, len(pool))
	for i := range counts {
		counts[i] = 1
	}
	fitLocations(pool, counts, m.locationTargets(pool, f))
}

func (m *FiddlesticksAiModelTheory4) unknownPool() []*CardLocationProbability {
	pool := []*CardLocationProbability{}
	for _, location := range m.cards {
		if location.unknown {
			pool = append(pool, location)
		}
	}
	return pool
}

func (m *FiddlesticksAiModelTheory4) locationTargets(pool []*CardLocationProbability, f *GameFiddlesticks) []LocationTarget {
	targets := []LocationTarget{{KITTY_COLUMN, float64(len(pool))}}
	for id, opponent := range m.opponents {
		if opponent == nil {
			continue
		}
		hand_size := float64(len(f.players[id].cards) - len(f.players[id].cards_played))
		targets = append(targets, LocationTarget{id, hand_size})
		targets[0].amount -= hand_size
	}
	return targets
}

func (m *FiddlesticksAiModelTheory4) weightByBids(pool []*CardLocationProbability, f *GameFiddlesticks) {
	average := m.averageStrength(pool, f)
	for id, opponent := range m.opponents {
		if opponent == nil || !f.players[id].has_bet {
			continue
		}
		for _, location := range pool {
			location.player[id] *= m.bidFactor(location.card, id, average, f)
		}
	}
}

func (m *FiddlesticksAiModelTheory4) averageStrength(pool []*CardLocationProbability, f *GameFiddlesticks) float64 {
	total := 0.0
	for _, location := range pool {
		total += (&FiddlesticksAiModelTheory1{}).probabilityToWin(location.card, f.trump.GetSuit(), len(f.players)-1)
	}
	return total / float64(len(pool))
}

func (m *FiddlesticksAiModelTheory4) bidFactor(card *game_utils.StandardCard, id int, average float64, f *GameFiddlesticks) float64 {
	hand_size := float64(len(f.players[id].cards) - len(f.players[id].cards_played))
	card_strength := (&FiddlesticksAiModelTheory1{}).probabilityToWin(card, f.trump.GetSuit(), len(f.players)-1)
	expected := hand_size*average + (card_strength - average)
	gap := (float64(int(f.players[id].bet)-int(f.players[id].tricks)) - expected) / m.bid_sigma
	return math.Exp(-0.5 * gap * gap * m.bid_trust)
}

func (m *FiddlesticksAiModelTheory4) Bet(p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	base_bet := m.FiddlesticksAiModelTheory3.Bet(p, f)
	return util.Clamp(base_bet+m.bet_optimism, 0, float64(f.round))
}

func (m *FiddlesticksAiModelTheory4) behavioralFactor(card *game_utils.StandardCard, f *GameFiddlesticks, tricks_needed_factor float64, prob_to_win float64) float64 {
	factor := 1.0
	rank_frac := float64(card.GetNumber()-2) / 12.0
	if m.trick_chasing != 0 {
		factor *= 1.0 + m.trick_chasing*tricks_needed_factor*prob_to_win
	}
	if m.trump_hoarding != 0 && f.trump != nil && card.GetSuit() == f.trump.GetSuit() {
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
func (m *FiddlesticksAiModelTheory4) CardWeights(p *FiddlesticksPlayer, f *GameFiddlesticks, valid_cards []int) []float64 {
	if m.locations_stale {
		m.updateLocationsWithBids(f)
		m.locations_stale = false
	}
	weights := m.FiddlesticksAiModelTheory3.CardWeights(p, f, valid_cards)
	if len(valid_cards) <= 1 {
		return weights
	}
	tricks_needed := int(p.bet) - int(p.tricks)
	tricks_left := len(p.cards) - len(p.cards_played)
	tricks_needed_factor := float64(tricks_needed) / float64(tricks_left)
	current_winning_card, _ := f.winningTrickCard()
	for i, card_index := range valid_cards {
		card := p.cards[card_index]
		prob_to_win := float64(0)
		if f.cardBeatsCard(card, current_winning_card) {
			prob_to_win = m.winProbability(card, p, f)
		}
		weights[i] *= m.behavioralFactor(card, f, tricks_needed_factor, prob_to_win)
		if m.vindictiveness > 0 && len(m.levels) == len(f.players) {
			weights[i] *= m.vindictivenessFactor(card, p, f)
		}
	}
	return weights
}

func (m *FiddlesticksAiModelTheory4) vindictivenessFactor(card *game_utils.StandardCard, p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	effect := m.opponentTrickEffect(card, p, f, m.levels)
	return math.Exp(util.Clamp(m.vindictiveness*effect, -LOGIT_SHIFT_LIMIT, LOGIT_SHIFT_LIMIT))
}
