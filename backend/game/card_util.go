package game

import "math/rand"

type Card struct {
}

type StandardCard struct {
	Card
	suit   uint8
	number uint8
}

func (c *StandardCard) getName() string {
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

type Deck interface {
	Size() int
	SizeDrawPile() int
	SizeDiscardPile() int
	Reset()
	ShuffleDrawPile()
	ShuffleDiscardPile()
	DrawCard() *StandardCard
	DealCards(players uint8, cards uint8) [][]*StandardCard
}

type StandardDeck struct {
	cards        [52]*StandardCard // position determines which card is which
	draw_pile    []*StandardCard
	discard_pile []*StandardCard
}

func CreateStandardDeck() *StandardDeck {
	deck := StandardDeck{}
	for i := 1; i <= 4; i++ {
		for j := 2; j <= 14; j++ {
			position := (i-1)*13 + (j - 2)
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

func (d *StandardDeck) Reset() {
	d.draw_pile = make([]*StandardCard, len(d.cards))
	d.discard_pile = []*StandardCard{}
	perm := rand.Perm(len(d.cards))
	for i, v := range perm {
		d.draw_pile[v] = d.cards[i]
	}
}

func (d *StandardDeck) ShuffleDrawPile() {
	new_draw_pile := make([]*StandardCard, len(d.draw_pile))
	perm := rand.Perm(len(d.draw_pile))
	for i, v := range perm {
		new_draw_pile[v] = d.draw_pile[i]
	}
	d.draw_pile = new_draw_pile
}

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

func (d *StandardDeck) DrawCard() *StandardCard {
	if len(d.draw_pile) == 0 {
		return nil
	}
	card := d.draw_pile[0]
	d.draw_pile = d.draw_pile[:len(d.draw_pile)-1]
	return card
}

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
