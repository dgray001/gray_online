package fiddlesticks

import "github.com/dgray001/gray_online/game/game_utils"

func opponentSign(f *GameFiddlesticks, id int) float64 {
	if f.players[id].tricks >= f.players[id].bet {
		return 1
	}
	return -1
}

func (m *FiddlesticksAiModelTheory3) opponentTrickEffect(card *game_utils.StandardCard, p *FiddlesticksPlayer, f *GameFiddlesticks, levels []float64) float64 {
	lead_suit := card.GetSuit()
	if len(f.trick) > 0 {
		lead_suit = f.trick[0].GetSuit()
	}
	effect := 0.0
	for i := 1; i < len(f.players)-len(f.trick); i++ {
		id := (p.player.Player_id + i) % len(f.players)
		effect += levels[id] * opponentSign(f, id) * (2*m.beatProbability(card, id, lead_suit, f) - 1)
	}
	if winning_card, index := f.winningTrickCard(); winning_card != nil {
		id := (f.trick_leader + index) % len(f.players)
		effect += levels[id] * opponentSign(f, id) * (2*m.holdProbability(card, winning_card, p, f, lead_suit) - 1)
	}
	return effect
}

func (m *FiddlesticksAiModelTheory3) holdProbability(card *game_utils.StandardCard, winning_card *game_utils.StandardCard, p *FiddlesticksPlayer, f *GameFiddlesticks, lead_suit uint8) float64 {
	if f.cardBeatsCard(card, winning_card) {
		return 0
	}
	hold := 1.0
	for i := 1; i < len(f.players)-len(f.trick); i++ {
		id := (p.player.Player_id + i) % len(f.players)
		hold *= 1 - m.beatProbability(winning_card, id, lead_suit, f)
	}
	return hold
}
