package fiddlesticks

import (
	"fmt"
	"github.com/dgray001/gray_online/game/game_utils"
)

func validateLearningHands(hands [][]*game_utils.StandardCard, trump *game_utils.StandardCard) error {
	hand := len(hands[0])
	if hand < 1 || hand*len(hands) > 51 {
		return fmt.Errorf("invalid hand size")
	}
	used := map[uint16]bool{trump.Hash(): true}
	for _, cards := range hands {
		if len(cards) != hand {
			return fmt.Errorf("hands must have equal size")
		}
		for _, card := range cards {
			if card == nil || !card.Valid() || used[card.Hash()] {
				return fmt.Errorf("invalid or repeated card")
			}
			used[card.Hash()] = true
		}
	}
	return nil
}
