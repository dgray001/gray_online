package fiddlesticks

func (m *FiddlesticksAiModelTheory4) explainPlay(p *FiddlesticksPlayer, f *GameFiddlesticks) DecisionExplanation {
	valid, _ := f.validCards(p.player.Player_id)
	weights := m.CardWeights(p, f, valid)
	total := 0.0
	for _, w := range weights {
		total += w
	}
	e := DecisionExplanation{Kind: "play-card", UnknownCards: m.unknown_cards, Temperature: m.temperature, Assumption: "Model4 belief estimates, contract utility, heuristic shift, and temperature-scaled softmax; these are not exact odds."}
	for i, index := range valid {
		card := p.cards[index]
		win := m.currentWinProbability(card, p, f)
		utility := m.evaluateCardQ(win, int(p.bet)-int(p.tricks), m.otherWinProbs(p, t4Slot(card), f), int(p.bet), f)
		e.Cards = append(e.Cards, ExplainedCard{Index: index, Suit: card.GetSuit(), Number: card.GetNumber(), WinProbability: win, Utility: utility, Heuristic: m.heuristicAdjustment(card, p, f), Probability: weights[i] / total})
	}
	return e
}
