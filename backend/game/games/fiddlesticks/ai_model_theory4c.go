package fiddlesticks

import (
	"math"
	"math/rand"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
)

type sampledWorld struct {
	hands  [][]*game_utils.StandardCard
	weight float64
}

type FiddlesticksAiModelTheory4c struct {
	FiddlesticksAiModelTheory3
	worlds         float64
	bid_sigma      float64
	paranoia       float64
	bid_trust      float64
	trick_urgency  float64
	vindictiveness float64
	levels         []float64
	strength       *FiddlesticksAiModelTheory1
}

func createFiddlesticksAiModelTheory4c(model_input map[string]string) *FiddlesticksAiModelTheory4c {
	m := &FiddlesticksAiModelTheory4c{worlds: 48, bid_sigma: 1, strength: &FiddlesticksAiModelTheory1{}}
	dials := map[string]*float64{"worlds": &m.worlds, "bid_sigma": &m.bid_sigma, "paranoia": &m.paranoia,
		"bid_trust": &m.bid_trust, "trick_urgency": &m.trick_urgency, "vindictiveness": &m.vindictiveness}
	base_input := map[string]string{}
	for k, v := range model_input {
		if dial, ok := dials[k]; ok {
			*dial = util.ParseFloat(v)
		} else {
			base_input[k] = v
		}
	}
	m.FiddlesticksAiModelTheory3 = *createFiddlesticksAiModelTheory3(base_input)
	return m
}

func (m *FiddlesticksAiModelTheory4c) ApplyUpdate(p *FiddlesticksPlayer, f *GameFiddlesticks, u *game.UpdateMessage) {
	m.FiddlesticksAiModelTheory3.ApplyUpdate(p, f, u)
	if u.Kind == "deal-round" && len(m.levels) != len(f.players) {
		m.levels = make([]float64, len(f.players))
		for i := range m.levels {
			m.levels[i] = 1
		}
	}
}

func (m *FiddlesticksAiModelTheory4c) sampleWorlds(p *FiddlesticksPlayer, f *GameFiddlesticks) []*sampledWorld {
	worlds := make([]*sampledWorld, int(m.worlds))
	for i := range worlds {
		hands := m.sampleHands(p, f)
		worlds[i] = &sampledWorld{hands: hands, weight: m.bidWeight(hands, f)}
	}
	guardWeights(worlds)
	return worlds
}

func (m *FiddlesticksAiModelTheory4c) sampleHands(p *FiddlesticksPlayer, f *GameFiddlesticks) [][]*game_utils.StandardCard {
	hands := make([][]*game_utils.StandardCard, len(f.players))
	taken := [52]bool{}
	for _, id := range rand.Perm(len(f.players)) {
		if id == p.player.Player_id {
			continue
		}
		for range len(f.players[id].cards) - len(f.players[id].cards_played) {
			card := m.drawCard(id, &taken)
			if card == nil {
				break
			}
			taken[cardSlot(card)] = true
			hands[id] = append(hands[id], card)
		}
	}
	return hands
}

func (m *FiddlesticksAiModelTheory4c) drawCard(id int, taken *[52]bool) *game_utils.StandardCard {
	total := 0.0
	for slot, location := range m.cards {
		if location.unknown && !taken[slot] {
			total += location.player[id]
		}
	}
	pick := rand.Float64() * total
	var chosen *game_utils.StandardCard
	for slot, location := range m.cards {
		if !location.unknown || taken[slot] || location.player[id] <= 0 {
			continue
		}
		chosen = location.card
		if pick -= location.player[id]; pick <= 0 {
			break
		}
	}
	return chosen
}

func (m *FiddlesticksAiModelTheory4c) bidWeight(hands [][]*game_utils.StandardCard, f *GameFiddlesticks) float64 {
	if m.bid_trust == 0 {
		return 1
	}
	exponent := 0.0
	for id, hand := range hands {
		if hand == nil || !f.players[id].has_bet {
			continue
		}
		strength := 0.0
		for _, card := range hand {
			strength += m.strength.probabilityToWin(card, f.trump.GetSuit(), len(f.players)-1)
		}
		gap := (strength - float64(f.players[id].bet)) / m.bid_sigma
		exponent -= 0.5 * gap * gap
	}
	return math.Exp(m.bid_trust * exponent)
}

func (m *FiddlesticksAiModelTheory4c) bestChallenger(hand []*game_utils.StandardCard, best *game_utils.StandardCard, lead_suit uint8, f *GameFiddlesticks) *game_utils.StandardCard {
	has_lead := false
	for _, card := range hand {
		has_lead = has_lead || card.GetSuit() == lead_suit
	}
	var challenger *game_utils.StandardCard
	for _, card := range hand {
		if has_lead && card.GetSuit() != lead_suit {
			continue
		}
		if f.cardBeatsCard(card, best) && (challenger == nil || f.cardBeatsCard(card, challenger)) {
			challenger = card
		}
	}
	return challenger
}

func (m *FiddlesticksAiModelTheory4c) trickWinner(card *game_utils.StandardCard, w *sampledWorld, p *FiddlesticksPlayer, f *GameFiddlesticks) int {
	winner, best := p.player.Player_id, card
	lead_suit := card.GetSuit()
	if len(f.trick) > 0 {
		lead_suit = f.trick[0].GetSuit()
	}
	if current, index := f.winningTrickCard(); !f.cardBeatsCard(card, current) {
		winner, best = (f.trick_leader+index)%len(f.players), current
	}
	for i := 1; i < len(f.players)-len(f.trick); i++ {
		id := (p.player.Player_id + i) % len(f.players)
		if challenger := m.bestChallenger(w.hands[id], best, lead_suit, f); challenger != nil {
			winner, best = id, challenger
		}
	}
	return winner
}

func (m *FiddlesticksAiModelTheory4c) worldHarm(winner int, p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	rivals := []int{}
	for i := 1; i < len(f.players)-len(f.trick); i++ {
		rivals = append(rivals, (p.player.Player_id+i)%len(f.players))
	}
	if _, index := f.winningTrickCard(); index >= 0 {
		rivals = append(rivals, (f.trick_leader+index)%len(f.players))
	}
	harm := 0.0
	for _, id := range rivals {
		outcome := -1.0
		if winner == id {
			outcome = 1
		}
		harm += m.levels[id] * opponentSign(f, id) * outcome
	}
	return harm
}

func (m *FiddlesticksAiModelTheory4c) evaluate(card *game_utils.StandardCard, worlds []*sampledWorld, p *FiddlesticksPlayer, f *GameFiddlesticks) (float64, float64) {
	total, wins, harm, always := 0.0, 0.0, 0.0, 1.0
	for _, w := range worlds {
		winner := m.trickWinner(card, w, p, f)
		total += w.weight
		if m.vindictiveness != 0 && len(m.levels) == len(f.players) {
			harm += w.weight * m.worldHarm(winner, p, f)
		}
		if winner == p.player.Player_id {
			wins += w.weight
		} else {
			always = 0
		}
	}
	return (1-m.paranoia)*wins/total + m.paranoia*always, harm / total
}

func (m *FiddlesticksAiModelTheory4c) CardWeights(p *FiddlesticksPlayer, f *GameFiddlesticks, valid_cards []int) []float64 {
	if len(valid_cards) == 1 {
		return []float64{1}
	}
	if m.locations_stale {
		m.updateLocations(f)
		m.locations_stale = false
	}
	worlds := m.sampleWorlds(p, f)
	needed := float64(int(p.bet)-int(p.tricks)) / float64(len(p.cards)-len(p.cards_played)) * (1 + m.trick_urgency)
	weighting := m.weighting(f)
	weights := make([]float64, len(valid_cards))
	for i, card_index := range valid_cards {
		card := p.cards[card_index]
		win, harm := m.evaluate(card, worlds, p, f)
		shift := util.Clamp(m.vindictiveness*harm, -LOGIT_SHIFT_LIMIT, LOGIT_SHIFT_LIMIT)
		weights[i] = weighting.calculateWeight(needed, win) * m.heuristicFactor(card, f, needed, win) * math.Exp(shift)
	}
	return weights
}

func (m *FiddlesticksAiModelTheory4c) weighting(f *GameFiddlesticks) *FiddlesticksAiModelTheory1 {
	seat_fraction := float64(len(f.trick)) / float64(len(f.players)-1)
	aggressive := util.Clamp(scaledByPlayers(m.aggressive_factor, m.aggressive_factor_slope, len(f.players)), 0.2, 5)
	late_seat := util.Clamp(scaledByPlayers(m.late_seat_bonus, m.late_seat_bonus_slope, len(f.players)), 0, 2)
	return &FiddlesticksAiModelTheory1{aggressive_factor: aggressive * (1 + late_seat*seat_fraction)}
}

func guardWeights(worlds []*sampledWorld) {
	sum, square := 0.0, 0.0
	for _, w := range worlds {
		sum += w.weight
		square += w.weight * w.weight
	}
	if square == 0 || sum*sum/square < 2 {
		for _, w := range worlds {
			w.weight = 1
		}
	}
}
