package deprecated

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
)

func (m *FiddlesticksAiModelTheory4c) updateLocations(f *GameFiddlesticks) {
	pool := []*t4cLocation{}
	for _, location := range m.cards {
		if location.unknown {
			location.markUnknown(1, 1)
			pool = append(pool, location)
		}
	}
	targets := []t4cTarget{{t4cKittyColumn, float64(len(pool))}}
	voided := false
	for id, opponent := range m.opponents {
		if opponent == nil {
			continue
		}
		hand_size := float64(len(f.players[id].cards) - len(f.players[id].cards_played))
		targets = append(targets, t4cTarget{id, hand_size})
		targets[0].amount -= hand_size
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
	t4cFitBySuit(pool, targets)
}

func (m *FiddlesticksAiModelTheory4c) beatProbability(card *game_utils.StandardCard, opponent_id int, lead_suit uint8, f *GameFiddlesticks) float64 {
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

func (m *FiddlesticksAiModelTheory4c) heuristicAdjustment(card *game_utils.StandardCard, p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	shift := 0.0
	if m.trump_hoarding != 0 && card.GetSuit() == f.trump.GetSuit() {
		needed := float64(int(p.bet) - int(p.tricks))
		urgency := util.Clamp(needed/float64(len(p.cards)-len(p.cards_played)), 0, 1)
		shift -= m.trump_hoarding * float64(card.GetNumber()-2) / 12.0 * (1.0 - urgency)
	}
	if m.void_hunting != 0 && len(f.trick) == 0 {
		for _, opp := range m.opponents {
			if opp != nil && opp.missing_suits[card.GetSuit()] {
				shift += m.void_hunting
				break
			}
		}
	}
	if m.vindictiveness != 0 {
		shift += m.vindictiveness * m.opponentTrickEffect(card, p, f)
	}
	return shift
}

func t4cOpponentSign(f *GameFiddlesticks, id int) float64 {
	if f.players[id].tricks >= f.players[id].bet {
		return 1
	}
	return -1
}

func (m *FiddlesticksAiModelTheory4c) opponentTrickEffect(card *game_utils.StandardCard, p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	lead_suit := card.GetSuit()
	if len(f.trick) > 0 {
		lead_suit = f.trick[0].GetSuit()
	}
	effect := 0.0
	for i := 1; i < len(f.players)-len(f.trick); i++ {
		id := (p.player.Player_id + i) % len(f.players)
		effect += m.levels[id] * t4cOpponentSign(f, id) * (2*m.beatProbability(card, id, lead_suit, f) - 1)
	}
	if winning_card, index := f.winningTrickCard(); winning_card != nil {
		id := (f.trick_leader + index) % len(f.players)
		effect += m.levels[id] * t4cOpponentSign(f, id) * (2*m.holdProbability(card, winning_card, p, f, lead_suit) - 1)
	}
	return effect
}

func (m *FiddlesticksAiModelTheory4c) holdProbability(card *game_utils.StandardCard, winning_card *game_utils.StandardCard, p *FiddlesticksPlayer, f *GameFiddlesticks, lead_suit uint8) float64 {
	if f.cardBeatsCard(card, winning_card) {
		return 0
	}
	hold := 1.0
	for i := 1; i < len(f.players)-len(f.trick); i++ {
		hold *= 1 - m.beatProbability(winning_card, (p.player.Player_id+i)%len(f.players), lead_suit, f)
	}
	return hold
}
