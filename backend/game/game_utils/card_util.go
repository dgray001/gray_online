package game_utils

import (
	"math/rand"

	"github.com/gin-gonic/gin"
)

type Card struct {
}

type StandardCard struct {
	Card
	suit   uint8
	number uint8
}

func (c *StandardCard) GetName() string {
	return c.getNumberName() + " of " + c.getSuitName() + "s"
}

func (c *StandardCard) getNumberName() string {
	switch c.number {
	case 2:
		return "Two"
	case 3:
		return "Three"
	case 4:
		return "Four"
	case 5:
		return "Five"
	case 6:
		return "Six"
	case 7:
		return "Seven"
	case 8:
		return "Eight"
	case 9:
		return "Nine"
	case 10:
		return "Ten"
	case 11:
		return "Jack"
	case 12:
		return "Queen"
	case 13:
		return "King"
	case 14:
		return "Ace"
	default:
		return "Error"
	}
}

func (c *StandardCard) getSuitName() string {
	switch c.suit {
	case 1:
		return "Diamond"
	case 2:
		return "Club"
	case 3:
		return "Heart"
	case 4:
		return "Spade"
	default:
		return "Error"
	}
}

func (c *StandardCard) GetSuit() uint8 {
	return c.suit
}

func (c *StandardCard) GetSuitColor() string {
	return SuitColor(c.suit)
}

func SuitColor(suit uint8) string {
	switch suit {
	case 1:
		return "Red"
	case 2:
		return "Black"
	case 3:
		return "Red"
	case 4:
		return "Black"
	default:
		return "Error"
	}
}

func (c *StandardCard) GetNumber() uint8 {
	return c.number
}

func (c *StandardCard) Valid() bool {
	if c.suit < 1 || c.suit > 4 {
		return false
	}
	if c.number < 2 || c.number > 14 {
		return false
	}
	return true
}

func (c *StandardCard) ToFrontend() gin.H {
	return gin.H{
		"suit":   c.suit,
		"number": c.number,
	}
}

type Deck interface {
	Size() int
	SizeDrawPile() int
	SizeDiscardPile() int
	Reset()
	ShuffleDrawPile()
	ShuffleDiscardPile()
	DrawCard() *StandardCard
	DealCards(players uint8, cards uint8) [][]*StandardCard
	Valid() bool
	ToFrontend() gin.H
}

type StandardDeckType int8

const (
	StandardDeckType_ERROR StandardDeckType = iota
	StandardDeckType_STANDARD52
	StandardDeckType_STANDARD24
)

type StandardDeck struct {
	deck_type    StandardDeckType
	cards        []*StandardCard // position determines which card is which
	draw_pile    []*StandardCard
	discard_pile []*StandardCard
}

func CreateStandardDeck() *StandardDeck {
	return CreateStandardDeckConfig(StandardDeckType_STANDARD52)
}

func CreateStandardDeckConfig(deck_type StandardDeckType) *StandardDeck {
	var cards = 0
	switch deck_type {
	case StandardDeckType_STANDARD52:
		cards = 52
	case StandardDeckType_STANDARD24:
		cards = 24
	}
	deck := StandardDeck{
		deck_type: deck_type,
		cards:     make([]*StandardCard, cards),
	}
	var suit_start = 1
	var suit_end = 4
	var value_start = 2
	var value_end = 14
	if deck_type == StandardDeckType_STANDARD24 {
		value_start = 9
	}
	for i := suit_start; i <= suit_end; i++ {
		for j := value_start; j <= value_end; j++ {
			position := (i-suit_start)*(value_end-value_start+1) + (j - value_start)
			deck.cards[position] = &StandardCard{suit: uint8(i), number: uint8(j)}
		}
	}
	deck.Reset()
	return &deck
}

func (d *StandardDeck) Size() int {
	return len(d.cards)
}

func (d *StandardDeck) SizeDrawPile() int {
	return len(d.draw_pile)
}

func (d *StandardDeck) SizeDiscardPile() int {
	return len(d.discard_pile)
}

// Shuffles all cards into the draw pile
func (d *StandardDeck) Reset() {
	d.draw_pile = make([]*StandardCard, len(d.cards))
	d.discard_pile = []*StandardCard{}
	perm := rand.Perm(len(d.cards))
	for i, v := range perm {
		d.draw_pile[v] = d.cards[i]
	}
}

// Shuffles current draw pile
func (d *StandardDeck) ShuffleDrawPile() {
	new_draw_pile := make([]*StandardCard, len(d.draw_pile))
	perm := rand.Perm(len(d.draw_pile))
	for i, v := range perm {
		new_draw_pile[v] = d.draw_pile[i]
	}
	d.draw_pile = new_draw_pile
}

// Shuffles current discard pile into current draw pile
func (d *StandardDeck) ShuffleDiscardPile() {
	new_draw_pile := make([]*StandardCard, len(d.draw_pile)+len(d.discard_pile))
	perm := rand.Perm(len(d.draw_pile) + len(d.discard_pile))
	for i, v := range perm {
		if i < len(d.draw_pile) {
			new_draw_pile[v] = d.draw_pile[i]
		} else {
			new_draw_pile[v] = d.discard_pile[i]
		}
	}
	d.draw_pile = new_draw_pile
}

// removes and returns top card of draw pile, or nil if draw pile is empty
func (d *StandardDeck) DrawCard() *StandardCard {
	if len(d.draw_pile) == 0 {
		return nil
	}
	card := d.draw_pile[0]
	d.draw_pile = d.draw_pile[1:]
	return card
}

// Draws cards from the deck into hands of specified size, or returns nil if not enough cards
func (d *StandardDeck) DealCards(players uint8, cards uint8) [][]*StandardCard {
	if int(players*cards) > len(d.draw_pile) {
		return nil
	}
	card_return := make([][]*StandardCard, 0)
	for player := 0; player < int(players); player++ {
		card_return = append(card_return, make([]*StandardCard, 0))
	}
	for card := 0; card < int(cards); card++ {
		for player := 0; player < int(players); player++ {
			card_return[player] = append(card_return[player], d.DrawCard())
		}
	}
	return card_return
}

func (d *StandardDeck) Valid() bool {
	for _, card := range d.cards {
		if card == nil || !card.Valid() {
			return false
		}
	}
	for _, card := range d.draw_pile {
		if card == nil || !card.Valid() {
			return false
		}
	}
	for _, card := range d.discard_pile {
		if card == nil || !card.Valid() {
			return false
		}
	}
	return true
}

func (d *StandardDeck) ToFrontend() gin.H {
	deck := gin.H{}
	cards := []gin.H{}
	for _, card := range d.cards {
		if card != nil {
			cards = append(cards, card.ToFrontend())
		}
	}
	deck["cards"] = cards
	draw_pile := []gin.H{}
	for _, card := range d.draw_pile {
		if card != nil {
			draw_pile = append(draw_pile, card.ToFrontend())
		}
	}
	deck["draw_pile"] = draw_pile
	discard_pile := []gin.H{}
	for _, card := range d.discard_pile {
		if card != nil {
			discard_pile = append(discard_pile, card.ToFrontend())
		}
	}
	deck["discard_pile"] = discard_pile
	return deck
}
