package fiddlesticks

import (
	"math"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
)

const LOGIT_SHIFT_LIMIT = 5.0

type FiddlesticksAiModelTheory4a struct {
	FiddlesticksAiModelTheory3
	bet_optimism        float64
	risk_tolerance      float64
	trump_conservation  float64
	contract_aggression float64
	vindictiveness      float64
	levels              []float64
}

func createFiddlesticksAiModelTheory4a(model_input map[string]string) *FiddlesticksAiModelTheory4a {
	m := &FiddlesticksAiModelTheory4a{}
	dials := map[string]*float64{"bet_optimism": &m.bet_optimism, "risk_tolerance": &m.risk_tolerance,
		"trump_conservation": &m.trump_conservation, "contract_aggression": &m.contract_aggression,
		"vindictiveness": &m.vindictiveness}
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

func (m *FiddlesticksAiModelTheory4a) ApplyUpdate(p *FiddlesticksPlayer, f *GameFiddlesticks, u *game.UpdateMessage) {
	m.FiddlesticksAiModelTheory3.ApplyUpdate(p, f, u)
	if u.Kind == "deal-round" && len(m.levels) != len(f.players) {
		m.levels = make([]float64, len(f.players))
		for i := range m.levels {
			m.levels[i] = 1
		}
	}
}

func (m *FiddlesticksAiModelTheory4a) Bet(p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	return util.Clamp(m.FiddlesticksAiModelTheory3.Bet(p, f)+m.bet_optimism, 0, float64(f.round))
}

func (m *FiddlesticksAiModelTheory4a) CardWeights(p *FiddlesticksPlayer, f *GameFiddlesticks, valid_cards []int) []float64 {
	weights := m.FiddlesticksAiModelTheory3.CardWeights(p, f, valid_cards)
	if len(valid_cards) == 1 {
		return weights
	}
	needed := util.Clamp(float64(int(p.bet)-int(p.tricks))/float64(len(p.cards)-len(p.cards_played)), 0, 1)
	winning_card, _ := f.winningTrickCard()
	for i, card_index := range valid_cards {
		card := p.cards[card_index]
		win := 0.0
		if f.cardBeatsCard(card, winning_card) {
			win = m.winProbability(card, p, f)
		}
		shift := util.Clamp(m.logitShift(card, win, needed, p, f), -LOGIT_SHIFT_LIMIT, LOGIT_SHIFT_LIMIT)
		weights[i] *= math.Exp(shift)
	}
	return weights
}

func (m *FiddlesticksAiModelTheory4a) logitShift(card *game_utils.StandardCard, win float64, needed float64, p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	rank_fraction := float64(card.GetNumber()-2) / 12
	shift := m.risk_tolerance*4*win*(1-win) + m.contract_aggression*needed*win
	if card.GetSuit() == f.trump.GetSuit() {
		shift -= m.trump_conservation * rank_fraction * (1 - needed)
	}
	if m.vindictiveness != 0 && len(m.levels) == len(f.players) {
		shift += m.vindictiveness * m.opponentTrickEffect(card, p, f, m.levels)
	}
	return shift
}
