package deprecated

import (
	"math"

	"github.com/dgray001/gray_online/game/game_utils"
)

func (m *FiddlesticksAiModelTheory4) unknownPool() []*t4Location {
	pool := []*t4Location{}
	for _, location := range m.cards {
		if location.unknown {
			pool = append(pool, location)
		}
	}
	return pool
}

func (m *FiddlesticksAiModelTheory4) locationTargets(pool []*t4Location, f *GameFiddlesticks) []t4Target {
	targets := []t4Target{{t4KittyColumn, float64(len(pool))}}
	for id, opponent := range m.opponents {
		if opponent == nil {
			continue
		}
		hand_size := float64(len(f.players[id].cards) - len(f.players[id].cards_played))
		targets = append(targets, t4Target{id, hand_size})
		targets[0].amount -= hand_size
	}
	return targets
}

func (m *FiddlesticksAiModelTheory4) updateLocations(f *GameFiddlesticks) {
	pool := m.unknownPool()
	for _, location := range pool {
		location.markUnknown(1, 1)
	}
	targets := m.locationTargets(pool, f)
	voided := false
	for id, opponent := range m.opponents {
		if opponent == nil {
			continue
		}
		for _, location := range pool {
			if opponent.missing_suits[location.card.GetSuit()] {
				location.player[id] = 0
				voided = true
			}
		}
	}
	if !voided {
		for _, location := range pool {
			for _, target := range targets {
				location.setColumn(target.column, target.amount/float64(len(pool)))
			}
		}
		return
	}
	t4FitBySuit(pool, targets)
}

func (m *FiddlesticksAiModelTheory4) updateLocationsWithBids(f *GameFiddlesticks) {
	m.updateLocations(f)
	pool := m.unknownPool()
	if m.bid_trust == 0 || len(pool) == 0 {
		return
	}
	average := m.averageStrength(pool, f)
	for id, opponent := range m.opponents {
		if opponent == nil || !f.players[id].has_bet {
			continue
		}
		for _, location := range pool {
			location.player[id] *= m.bidFactor(location.card, id, average, f)
		}
	}
	counts := make([]float64, len(pool))
	for i := range counts {
		counts[i] = 1
	}
	t4Fit(pool, counts, m.locationTargets(pool, f))
}

func (m *FiddlesticksAiModelTheory4) averageStrength(pool []*t4Location, f *GameFiddlesticks) float64 {
	total := 0.0
	for _, location := range pool {
		total += t4ProbabilityToWin(location.card, f.trump.GetSuit(), len(f.players)-1)
	}
	return total / float64(len(pool))
}

func (m *FiddlesticksAiModelTheory4) bidFactor(card *game_utils.StandardCard, id int, average float64, f *GameFiddlesticks) float64 {
	hand_size := float64(len(f.players[id].cards) - len(f.players[id].cards_played))
	card_strength := t4ProbabilityToWin(card, f.trump.GetSuit(), len(f.players)-1)
	expected := hand_size*average + (card_strength - average)
	gap := (float64(int(f.players[id].bet)-int(f.players[id].tricks)) - expected) / m.bid_sigma
	return math.Exp(-0.5 * gap * gap * m.bid_trust)
}

func (m *FiddlesticksAiModelTheory4) winProbability(card *game_utils.StandardCard, p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	lead_suit := card.GetSuit()
	if len(f.trick) > 0 {
		lead_suit = f.trick[0].GetSuit()
	}
	probability := 1.0
	for i := 1; i < len(f.players)-len(f.trick); i++ {
		probability *= 1 - m.beatProbability(card, (p.player.Player_id+i)%len(f.players), lead_suit, f)
	}
	return probability
}

func (m *FiddlesticksAiModelTheory4) beatProbability(card *game_utils.StandardCard, opponent_id int, lead_suit uint8, f *GameFiddlesticks) float64 {
	var lead_total, lead_beating, other_total, other_beating float64
	no_lead := 1.0
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
	beat_probability := 0.0
	if lead_total > 0 {
		beat_probability += (1 - no_lead) * lead_beating / lead_total
	}
	if other_total > 0 {
		beat_probability += no_lead * other_beating / other_total
	}
	return beat_probability
}

func t4OpponentSign(f *GameFiddlesticks, id int) float64 {
	if f.players[id].tricks >= f.players[id].bet {
		return 1
	}
	return -1
}

func (m *FiddlesticksAiModelTheory4) opponentTrickEffect(card *game_utils.StandardCard, p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	lead_suit := card.GetSuit()
	if len(f.trick) > 0 {
		lead_suit = f.trick[0].GetSuit()
	}
	effect := 0.0
	for i := 1; i < len(f.players)-len(f.trick); i++ {
		id := (p.player.Player_id + i) % len(f.players)
		effect += m.levels[id] * t4OpponentSign(f, id) * (2*m.beatProbability(card, id, lead_suit, f) - 1)
	}
	if winning_card, index := f.winningTrickCard(); winning_card != nil {
		id := (f.trick_leader + index) % len(f.players)
		effect += m.levels[id] * t4OpponentSign(f, id) * (2*m.holdProbability(card, winning_card, p, f, lead_suit) - 1)
	}
	return effect
}

func (m *FiddlesticksAiModelTheory4) holdProbability(card *game_utils.StandardCard, winning_card *game_utils.StandardCard, p *FiddlesticksPlayer, f *GameFiddlesticks, lead_suit uint8) float64 {
	if f.cardBeatsCard(card, winning_card) {
		return 0
	}
	hold := 1.0
	for i := 1; i < len(f.players)-len(f.trick); i++ {
		hold *= 1 - m.beatProbability(winning_card, (p.player.Player_id+i)%len(f.players), lead_suit, f)
	}
	return hold
}
