package deprecated

import (
	"slices"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
)

type trickRollout struct {
	f     *GameFiddlesticks
	hands [][]*game_utils.StandardCard
	need  []int
}

func (r *trickRollout) strength(card *game_utils.StandardCard) int {
	if card.GetSuit() == r.f.trump.GetSuit() {
		return 100 + int(card.GetNumber())
	}
	return int(card.GetNumber())
}
func (r *trickRollout) preference(card *game_utils.StandardCard, best *game_utils.StandardCard, wantWin bool) int {
	strength := r.strength(card)
	switch {
	case best == nil && wantWin:
		return strength
	case best == nil:
		return -strength
	case r.f.cardBeatsCard(card, best) != wantWin:
		return -strength
	case wantWin:
		return 1000 - strength
	default:
		return 1000 + strength
	}
}
func (r *trickRollout) play(id int, best *game_utils.StandardCard, lead_suit uint8) *game_utils.StandardCard {
	hand, chosen := r.hands[id], -1
	for i, card := range hand {
		if best != nil && card.GetSuit() != lead_suit && hasSuit(hand, lead_suit) {
			continue
		}
		if chosen < 0 || r.preference(card, best, r.need[id] > 0) > r.preference(hand[chosen], best, r.need[id] > 0) {
			chosen = i
		}
	}
	if chosen < 0 {
		return nil
	}
	card := hand[chosen]
	r.hands[id] = slices.Delete(hand, chosen, chosen+1)
	return card
}
func (r *trickRollout) playTrick(order []int, best *game_utils.StandardCard, winner int, lead_suit uint8) int {
	for _, id := range order {
		card := r.play(id, best, lead_suit)
		if card == nil {
			continue
		}
		if best == nil {
			lead_suit = card.GetSuit()
		}
		if best == nil || r.f.cardBeatsCard(card, best) {
			best, winner = card, id
		}
	}
	r.need[winner]--
	return winner
}
func (r *trickRollout) seats(first int, count int) []int {
	order := make([]int, count)
	for i := range order {
		order[i] = (first + i) % len(r.f.players)
	}
	return order
}
func (r *trickRollout) playCurrentTrick(card *game_utils.StandardCard, me int) int {
	best, index := r.f.winningTrickCard()
	winner, lead_suit := me, card.GetSuit()
	if best != nil {
		winner, lead_suit = (r.f.trick_leader+index)%len(r.f.players), r.f.trick[0].GetSuit()
	}
	if r.f.cardBeatsCard(card, best) {
		best, winner = card, me
	}
	return r.playTrick(r.seats(me+1, len(r.f.players)-len(r.f.trick)-1), best, winner, lead_suit)
}
func (m *FiddlesticksAiModelTheory4c) rolloutGap(cardIndex int, w *sampledWorld, p *FiddlesticksPlayer, f *GameFiddlesticks) int {
	me := p.player.Player_id
	r := &trickRollout{f: f, hands: make([][]*game_utils.StandardCard, len(f.players)), need: make([]int, len(f.players))}
	for id, player := range f.players {
		r.need[id] = int(player.bet) - int(player.tricks)
		r.hands[id] = slices.Clone(w.hands[id])
	}
	for i, card := range p.cards {
		if i != cardIndex && !util.Contains(p.cards_played, i) {
			r.hands[me] = append(r.hands[me], card)
		}
	}
	winner := r.playCurrentTrick(p.cards[cardIndex], me)
	for len(r.hands[me]) > 0 {
		winner = r.playTrick(r.seats(winner, len(f.players)), nil, winner, 0)
	}
	return r.need[me]
}
func hasSuit(hand []*game_utils.StandardCard, suit uint8) bool {
	return slices.ContainsFunc(hand, func(card *game_utils.StandardCard) bool { return card.GetSuit() == suit })
}
