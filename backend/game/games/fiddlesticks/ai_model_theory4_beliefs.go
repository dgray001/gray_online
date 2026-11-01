package fiddlesticks

import (
	"fmt"
	"math"
	"os"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/gin-gonic/gin"
)

const t4KittyColumn = -1
const t4FittingRounds = 20
const t4FittingTolerance = 1e-6

type t4Target struct {
	column int
	amount float64
}

type t4Location struct {
	card      *game_utils.StandardCard
	unknown   bool
	kitty     float64
	player    []float64 // indexed by player id
	opponents []int
}

func (l *t4Location) reset(players []*FiddlesticksPlayer, model_player int) {
	l.unknown = false
	l.kitty = 0
	l.player = make([]float64, len(players))
	l.opponents = l.opponents[:0]
	for i := range players {
		if i != model_player {
			l.opponents = append(l.opponents, i)
		}
	}
}
func (l *t4Location) markUnknown(prob_kitty float64, prob_hand float64) {
	l.unknown = true
	l.kitty = prob_kitty
	for _, id := range l.opponents {
		l.player[id] = prob_hand
	}
}
func (l *t4Location) markPlayed() {
	l.unknown = false
	l.kitty = 0
	for _, id := range l.opponents {
		l.player[id] = 0
	}
}
func (l *t4Location) column(id int) float64 {
	if id == t4KittyColumn {
		return l.kitty
	}
	return l.player[id]
}
func (l *t4Location) setColumn(id int, value float64) {
	if id == t4KittyColumn {
		l.kitty = value
		return
	}
	l.player[id] = value
}
func (l *t4Location) scaleColumn(id int, factor float64) {
	if id == t4KittyColumn {
		l.kitty *= factor
		return
	}
	l.player[id] *= factor
}
func (l *t4Location) copyFrom(other *t4Location) {
	l.kitty = other.kitty
	copy(l.player, other.player)
}
func (l *t4Location) normalize() {
	total := l.kitty
	for _, id := range l.opponents {
		total += l.player[id]
	}
	if total == 0 {
		return
	}
	l.kitty /= total
	for _, id := range l.opponents {
		l.player[id] /= total
	}
}

type t4HandCard struct {
	loseable_cards [52]bool
	loseable_count int
}

func (d *t4HandCard) addLoseCard(slot int) {
	if !d.loseable_cards[slot] {
		d.loseable_cards[slot] = true
		d.loseable_count++
	}
}
func (d *t4HandCard) cardPlayed(slot int) {
	if d.loseable_cards[slot] {
		d.loseable_cards[slot] = false
		d.loseable_count--
	}
}

type t4Opponent struct {
	missing_suits [5]bool
}

func t4Slot(card *game_utils.StandardCard) int {
	return int(card.GetSuit()-1)*13 + int(card.GetNumber()-2)
}

func (m *FiddlesticksAiModelTheory4) initCards() {
	for _, card := range game_utils.CreateStandardDeck().GetCards() {
		if card != nil {
			m.cards[t4Slot(card)] = &t4Location{card: card}
		}
	}
}

func (m *FiddlesticksAiModelTheory4) ApplyUpdate(p *FiddlesticksPlayer, f *GameFiddlesticks, u *game.UpdateMessage) {
	switch u.Kind {
	case "deal-round":
		m.dealRound(p, f)
	case "bet":
		m.locations_stale = true
	case "play-card":
		m.cardPlayed(u)
	default:
		fmt.Fprintln(os.Stderr, "Unknown update kind for theory model 4b:", u.Kind)
	}
}
func (m *FiddlesticksAiModelTheory4) dealRound(p *FiddlesticksPlayer, f *GameFiddlesticks) {
	if len(m.opponents) != len(f.players) {
		m.opponents = make([]*t4Opponent, len(f.players))
		for i := range f.players {
			if i != p.player.Player_id {
				m.opponents[i] = &t4Opponent{}
			}
		}
	}
	for _, opponent := range m.opponents {
		if opponent != nil {
			opponent.missing_suits = [5]bool{}
		}
	}
	if len(m.levels) != len(f.players) {
		m.levels = make([]float64, len(f.players))
		for i := range m.levels {
			m.levels[i] = 1.0
		}
	}
	m.hand = [52]*t4HandCard{}
	for _, card := range p.cards {
		m.hand[t4Slot(card)] = &t4HandCard{}
	}
	m.unknown_cards = 52 - 1 - int(f.round)
	m.base_prob_kitty = float64(52-1-len(f.players)*int(f.round)) / float64(m.unknown_cards)
	m.base_prob_hand = float64(f.round) / float64(m.unknown_cards)
	m.locations_stale = false
	for _, location := range m.cards {
		slot := t4Slot(location.card)
		location.reset(f.players, p.player.Player_id)
		if slot == t4Slot(f.trump) || m.hand[slot] != nil {
			continue
		}
		location.markUnknown(m.base_prob_kitty, m.base_prob_hand)
		for _, card := range p.cards {
			if f.cardBeatsCard(location.card, card) {
				m.hand[t4Slot(card)].addLoseCard(slot)
			}
		}
	}
}
func (m *FiddlesticksAiModelTheory4) cardPlayed(u *game.UpdateMessage) {
	suit := u.Content["card"].(gin.H)["suit"].(uint8)
	lead_suit := u.Content["lead_suit"].(uint8)
	number := u.Content["card"].(gin.H)["number"].(uint8)
	slot := t4Slot(game_utils.StandardCardStatic.CreateCard(suit, number))
	m.cards[slot].markPlayed()
	if m.hand[slot] != nil {
		m.hand[slot] = nil
	} else {
		for _, details := range m.hand {
			if details != nil {
				details.cardPlayed(slot)
			}
		}
		m.unknown_cards--
	}
	if opponent := m.opponents[u.Content["player_id"].(int)]; opponent != nil && suit != lead_suit {
		opponent.missing_suits[lead_suit] = true
	}
	m.locations_stale = true
}

func t4Fit(pool []*t4Location, counts []float64, targets []t4Target) {
	previous := make([]float64, 0, len(pool)*(len(targets)+1))
	for range t4FittingRounds {
		for _, location := range pool {
			location.normalize()
		}
		if t4Settled(pool, previous) {
			break
		}
		previous = previous[:0]
		for _, location := range pool {
			previous = append(previous, location.kitty)
			previous = append(previous, location.player...)
		}
		for _, target := range targets {
			t4ScaleToTarget(pool, counts, target.column, target.amount)
		}
	}
	for _, location := range pool {
		location.normalize()
	}
}
func t4Settled(pool []*t4Location, previous []float64) bool {
	if len(previous) == 0 {
		return false
	}
	i := 0
	for _, location := range pool {
		if math.Abs(location.kitty-previous[i]) > t4FittingTolerance {
			return false
		}
		i++
		for _, value := range location.player {
			if math.Abs(value-previous[i]) > t4FittingTolerance {
				return false
			}
			i++
		}
	}
	return true
}
func t4ScaleToTarget(pool []*t4Location, counts []float64, id int, target float64) {
	column := 0.0
	for i, location := range pool {
		column += counts[i] * location.column(id)
	}
	if column == 0 {
		return
	}
	for _, location := range pool {
		location.scaleColumn(id, target/column)
	}
}
func t4FitBySuit(pool []*t4Location, targets []t4Target) {
	var suit_cards [5][]*t4Location
	for _, location := range pool {
		suit_cards[location.card.GetSuit()] = append(suit_cards[location.card.GetSuit()], location)
	}
	representatives := make([]*t4Location, 0, 4)
	counts := make([]float64, 0, 4)
	for _, cards := range suit_cards {
		if len(cards) > 0 {
			representatives = append(representatives, cards[0])
			counts = append(counts, float64(len(cards)))
		}
	}
	t4Fit(representatives, counts, targets)
	for _, cards := range suit_cards {
		for i := 1; i < len(cards); i++ {
			cards[i].copyFrom(cards[0])
		}
	}
}
