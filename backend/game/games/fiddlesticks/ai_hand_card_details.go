package fiddlesticks

import "github.com/dgray001/gray_online/game/game_utils"

type HandCardDetails struct {
	// card
	card *game_utils.StandardCard
	// cards this card can beat in a trick
	beatable_cards map[game_utils.StandardCardHash]bool
	// cards this card can lose to in a trick
	loseable_cards map[game_utils.StandardCardHash]bool
}

func createHandCardDetails(card *game_utils.StandardCard) *HandCardDetails {
	details := &HandCardDetails{
		card:           card,
		beatable_cards: make(map[uint16]bool),
		loseable_cards: make(map[uint16]bool),
	}
	return details
}

func (d *HandCardDetails) addWinCard(hash game_utils.StandardCardHash) {
	d.beatable_cards[hash] = true
}

func (d *HandCardDetails) addLoseCard(hash game_utils.StandardCardHash) {
	d.loseable_cards[hash] = true
}

func (d *HandCardDetails) cardPlayed(hash game_utils.StandardCardHash) {
	delete(d.beatable_cards, hash)
	delete(d.loseable_cards, hash)
}
