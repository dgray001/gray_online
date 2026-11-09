package fiddlesticks

import "math"

func (m *FiddlesticksAiModelTheory4) explainBid(p *FiddlesticksPlayer, f *GameFiddlesticks) DecisionExplanation {
	e := DecisionExplanation{Kind: "bet", RawBid: m.Bet(p, f), Scale: m.bet_scale, Optimism: m.bet_optimism, UnknownCards: m.unknown_cards, Assumption: "Independent-card heuristic; actual tricks are correlated."}
	e.BidProbabilities = make([]float64, int(f.round)+1)
	floor := int(e.RawBid)
	fraction := e.RawBid - math.Floor(e.RawBid)
	e.BidProbabilities[floor] = 1 - fraction
	if floor+1 < len(e.BidProbabilities) {
		e.BidProbabilities[floor+1] = fraction
	}
	minimum := map[uint8]float64{}
	probabilities := []float64{}
	for index, card := range p.cards {
		base := math.Pow(1-float64(m.hand[t4Slot(card)].loseable_count)/float64(m.unknown_cards), float64(len(f.players)-1))
		win := m.winVsAllP(base, card.GetSuit() == f.trump.GetSuit())
		old, exists := minimum[card.GetSuit()]
		if !exists || win < old {
			minimum[card.GetSuit()] = win
		}
		e.Cards = append(e.Cards, ExplainedCard{Index: index, Suit: card.GetSuit(), Number: card.GetNumber(), WinProbability: win})
	}
	for i := range e.Cards {
		c := &e.Cards[i]
		c.Contribution = c.WinProbability*m.min_win_weight_factor + minimum[c.Suit]*(1-m.min_win_weight_factor)
		probabilities = append(probabilities, c.Contribution)
	}
	e.TrickProbabilities = t4Pmf(probabilities)
	if f.round == 1 {
		win, unknown, beat := f.oneCardOdds(p)
		e.TrickProbabilities, e.UnknownCards, e.BeatingCards, e.Assumption = []float64{1 - win, win}, unknown, beat, "Exact one-card odds assuming uniformly distributed unknown cards; other bids are not treated as evidence."
	}
	return e
}
