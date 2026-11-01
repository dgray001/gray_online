package deprecated

import (
	"math"
	"math/rand"

	"github.com/dgray001/gray_online/game/game_utils"
)

func (m *FiddlesticksAiModelTheory4c) sampleWorlds(p *FiddlesticksPlayer, f *GameFiddlesticks) []*sampledWorld {
	worlds := make([]*sampledWorld, max(1, int(m.worlds)))
	log_weights := make([]float64, len(worlds))
	for i := range worlds {
		hands, log_proposal := m.sampleHands(p, f)
		worlds[i] = &sampledWorld{hands: hands}
		log_weights[i] = m.bidLogLikelihood(hands, f) - log_proposal
	}
	normalizeWeights(worlds, log_weights)
	guardWeights(worlds)
	return worlds
}

func normalizeWeights(worlds []*sampledWorld, log_weights []float64) {
	peak := math.Inf(-1)
	for _, log_weight := range log_weights {
		peak = math.Max(peak, log_weight)
	}
	for i, w := range worlds {
		if !math.IsInf(peak, -1) {
			w.weight = math.Exp(log_weights[i] - peak)
		}
	}
}

func guardWeights(worlds []*sampledWorld) {
	sum, square := 0.0, 0.0
	for _, w := range worlds {
		sum += w.weight
		square += w.weight * w.weight
	}
	if square == 0 || sum*sum/square < 2.0 {
		for _, w := range worlds {
			w.weight = 1.0
		}
	}
}

func (m *FiddlesticksAiModelTheory4c) sampleHands(p *FiddlesticksPlayer, f *GameFiddlesticks) ([][]*game_utils.StandardCard, float64) {
	hands := make([][]*game_utils.StandardCard, len(f.players))
	taken := [52]bool{}
	log_proposal := 0.0
	for id := range f.players {
		if id == p.player.Player_id {
			continue
		}
		for range len(f.players[id].cards) - len(f.players[id].cards_played) {
			card, probability := m.drawCard(id, &taken)
			if card == nil {
				return hands, math.Inf(1)
			}
			taken[t4cSlot(card)] = true
			hands[id] = append(hands[id], card)
			log_proposal += math.Log(probability)
		}
	}
	return hands, log_proposal
}

func (m *FiddlesticksAiModelTheory4c) drawCard(id int, taken *[52]bool) (*game_utils.StandardCard, float64) {
	total := 0.0
	for slot, location := range m.cards {
		if location.unknown && !taken[slot] {
			total += location.player[id]
		}
	}
	pick := rand.Float64() * total
	var chosen *game_utils.StandardCard
	chosen_weight := 0.0
	for slot, location := range m.cards {
		if !location.unknown || taken[slot] || location.player[id] <= 0 {
			continue
		}
		chosen, chosen_weight = location.card, location.player[id]
		if pick -= chosen_weight; pick <= 0 {
			break
		}
	}
	return chosen, chosen_weight / total
}

func (m *FiddlesticksAiModelTheory4c) bidLogLikelihood(hands [][]*game_utils.StandardCard, f *GameFiddlesticks) float64 {
	if m.bid_trust == 0 {
		return 0.0
	}
	exponent := 0.0
	for id, hand := range hands {
		if hand == nil || !f.players[id].has_bet {
			continue
		}
		strength := 0.0
		for _, card := range hand {
			strength += t4cProbabilityToWin(card, f.trump.GetSuit(), len(f.players)-1)
		}
		gap := (strength - float64(int(f.players[id].bet)-int(f.players[id].tricks))) / m.bid_sigma
		exponent -= 0.5 * gap * gap
	}
	return m.bid_trust * exponent
}

func t4cProbabilityToWin(card *game_utils.StandardCard, trump_suit uint8, num_players int) float64 {
	cards_that_can_beat_it := game_utils.StandardCardStatic.AceValue() - card.GetNumber()
	if card.GetSuit() != trump_suit {
		cards_that_can_beat_it += 12
	}
	return math.Pow(1-float64(cards_that_can_beat_it)/float64(50), float64(num_players))
}
