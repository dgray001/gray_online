package fiddlesticks

func withoutReplacement(good int, total int, count int) float64 {
	if good < count {
		return 0
	}
	probability := 1.0
	for i := 0; i < count; i++ {
		probability *= float64(good-i) / float64(total-i)
	}
	return probability
}

func (f *GameFiddlesticks) oneCardOdds(p *FiddlesticksPlayer) (float64, int, int) {
	own := p.cards[0]
	total, beating, lower := 0, 0, 0
	for _, card := range f.deck.GetCards() {
		if card.Hash() == own.Hash() || card.Hash() == f.trump.Hash() {
			continue
		}
		total++
		if f.cardBeatsCard(card, own) {
			beating++
		}
		if card.GetSuit() == own.GetSuit() && card.GetNumber() < own.GetNumber() {
			lower++
		}
	}
	if p.player.Player_id == (f.dealer+1)%len(f.players) || own.GetSuit() == f.trump.GetSuit() {
		return withoutReplacement(total-beating, total, len(f.players)-1), total, beating
	}
	return float64(lower) / float64(total) * withoutReplacement(total-beating-1, total-1, len(f.players)-2), total, beating
}
