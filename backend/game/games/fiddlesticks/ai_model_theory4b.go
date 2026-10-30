package fiddlesticks

import (
	"math"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
)

type FiddlesticksAiModelTheory4b struct {
	FiddlesticksAiModelTheory3
	score_weight  float64
	bet_cushion   float64
	sandbagging   float64
	trump_control float64
	spite_weight  float64
	levels        []float64
}

func createFiddlesticksAiModelTheory4b(model_input map[string]string) *FiddlesticksAiModelTheory4b {
	m := &FiddlesticksAiModelTheory4b{}
	dials := map[string]*float64{"score_weight": &m.score_weight, "bet_cushion": &m.bet_cushion,
		"sandbagging": &m.sandbagging, "trump_control": &m.trump_control, "spite_weight": &m.spite_weight}
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

func (m *FiddlesticksAiModelTheory4b) ApplyUpdate(p *FiddlesticksPlayer, f *GameFiddlesticks, u *game.UpdateMessage) {
	m.FiddlesticksAiModelTheory3.ApplyUpdate(p, f, u)
	if u.Kind == "deal-round" && len(m.levels) != len(f.players) {
		m.levels = make([]float64, len(f.players))
		for i := range m.levels {
			m.levels[i] = 1
		}
	}
}

func (m *FiddlesticksAiModelTheory4b) Bet(p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	return util.Clamp(m.FiddlesticksAiModelTheory3.Bet(p, f)-m.bet_cushion, 0, float64(f.round))
}

func (m *FiddlesticksAiModelTheory4b) CardWeights(p *FiddlesticksPlayer, f *GameFiddlesticks, valid_cards []int) []float64 {
	weights := m.FiddlesticksAiModelTheory3.CardWeights(p, f, valid_cards)
	if len(valid_cards) == 1 {
		return weights
	}
	needed := int(p.bet) - int(p.tricks)
	remaining := len(p.cards) - len(p.cards_played)
	future_rate := m.futureWinRate(f)
	winning_card, _ := f.winningTrickCard()
	for i, card_index := range valid_cards {
		card := p.cards[card_index]
		win := 0.0
		if f.cardBeatsCard(card, winning_card) {
			win = m.winProbability(card, p, f)
		}
		contract := contractProbability(win, needed, remaining-1, future_rate)
		shift := m.score_weight*contract*m.pointsAtStake(p, f) + m.dialShift(card, win, needed, remaining, p, f)
		weights[i] *= math.Exp(util.Clamp(shift, -LOGIT_SHIFT_LIMIT, LOGIT_SHIFT_LIMIT))
	}
	return weights
}

func (m *FiddlesticksAiModelTheory4b) pointsAtStake(p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	return (float64(f.round_points) + float64(f.trick_points)*float64(p.bet)) / math.Max(1, float64(f.round_points))
}

func binomialPmf(k int, n int, q float64) float64 {
	if k < 0 || k > n {
		return 0
	}
	coefficient := 1.0
	for i := 1; i <= k; i++ {
		coefficient *= float64(n-k+i) / float64(i)
	}
	return coefficient * math.Pow(q, float64(k)) * math.Pow(1-q, float64(n-k))
}

func contractProbability(win float64, needed int, tricks_after int, rate float64) float64 {
	return win*binomialPmf(needed-1, tricks_after, rate) + (1-win)*binomialPmf(needed, tricks_after, rate)
}

func (m *FiddlesticksAiModelTheory4b) futureWinRate(f *GameFiddlesticks) float64 {
	total, count := 0.0, 0
	for slot, details := range m.hand {
		if details == nil {
			continue
		}
		lose := float64(details.loseable_count) / float64(m.unknown_cards)
		is_trump := m.cards[slot].card.GetSuit() == f.trump.GetSuit()
		total += m.winVsAllP(math.Pow(1-lose, float64(len(f.players)-1)), is_trump)
		count++
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func (m *FiddlesticksAiModelTheory4b) dialShift(card *game_utils.StandardCard, win float64, needed int, remaining int, p *FiddlesticksPlayer, f *GameFiddlesticks) float64 {
	rank_fraction := float64(card.GetNumber()-2) / 12
	shift := 0.0
	if needed <= 0 {
		shift -= m.sandbagging * win
	}
	if card.GetSuit() == f.trump.GetSuit() {
		urgency := util.Clamp(float64(needed)/float64(remaining), 0, 1)
		shift -= m.trump_control * rank_fraction * (1 - urgency)
	}
	if m.spite_weight != 0 && len(m.levels) == len(f.players) {
		shift += m.spite_weight * m.opponentTrickEffect(card, p, f, m.levels)
	}
	return shift
}
