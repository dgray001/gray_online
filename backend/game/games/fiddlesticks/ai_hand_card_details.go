package fiddlesticks

import "github.com/dgray001/gray_online/game/game_utils"

type HandCardDetails struct {
	// card
	card *game_utils.StandardCard
	// cards this card can beat in a trick, indexed by cardSlot
	beatable_cards [52]bool
	// cards this card can lose to in a trick, indexed by cardSlot
	loseable_cards [52]bool
	// number of true entries in loseable_cards
	loseable_count int
}

func createHandCardDetails(card *game_utils.StandardCard) *HandCardDetails {
	details := &HandCardDetails{card: card}
	return details
}

func (d *HandCardDetails) addWinCard(slot int) {
	d.beatable_cards[slot] = true
}

func (d *HandCardDetails) addLoseCard(slot int) {
	if !d.loseable_cards[slot] {
		d.loseable_cards[slot] = true
		d.loseable_count++
	}
}

func (d *HandCardDetails) cardPlayed(slot int) {
	d.beatable_cards[slot] = false
	if d.loseable_cards[slot] {
		d.loseable_cards[slot] = false
		d.loseable_count--
	}
}
