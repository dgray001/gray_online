package goldset

import (
	"fmt"
	"github.com/dgray001/gray_online/game/game_utils"
)

func reserveFoundationCard(hands [][]*game_utils.StandardCard, used map[uint16]int, seat int, card Card) error {
	if seat < 0 || seat >= len(hands) {
		return fmt.Errorf("invalid actor")
	}
	c := game_utils.StandardCardStatic.CreateCard(card.Suit, card.Number)
	if !c.Valid() {
		return fmt.Errorf("invalid card")
	}
	if owner, exists := used[c.Hash()]; exists {
		if owner != seat {
			return fmt.Errorf("card assigned to multiple players or face-up trump")
		}
		return nil
	}
	used[c.Hash()] = seat
	hands[seat] = append(hands[seat], c)
	return nil
}

func foundationVoids(spec FoundationCase) [][4]bool {
	result := make([][4]bool, spec.Players)
	lead := uint8(0)
	for index, play := range spec.Plays {
		if index%spec.Players == 0 {
			lead = play.Card.Suit
		}
		if play.Card.Suit != lead {
			result[play.Seat][lead-1] = true
		}
	}
	return result
}
