package goldset

import (
	"fmt"
	"github.com/dgray001/gray_online/game/game_utils"
	"math/rand"
)

func completeFoundationHands(spec FoundationCase, seed int64) ([][]*game_utils.StandardCard, error) {
	hands := make([][]*game_utils.StandardCard, spec.Players)
	used := map[uint16]int{}
	trump := game_utils.StandardCardStatic.CreateCard(spec.Trump.Suit, spec.Trump.Number)
	used[trump.Hash()] = -1
	for _, card := range spec.Hand {
		if err := reserveFoundationCard(hands, used, spec.Seat, card); err != nil {
			return nil, err
		}
	}
	if len(hands[spec.Seat]) != len(spec.Hand) {
		return nil, fmt.Errorf("duplicate own card")
	}
	for _, play := range spec.Plays {
		if err := reserveFoundationCard(hands, used, play.Seat, play.Card); err != nil {
			return nil, err
		}
	}
	for _, hand := range hands {
		if len(hand) > len(spec.Hand) {
			return nil, fmt.Errorf("script needs too many cards")
		}
	}
	pool := []*game_utils.StandardCard{}
	for _, card := range game_utils.CreateStandardDeck().GetCards() {
		if _, exists := used[card.Hash()]; !exists {
			pool = append(pool, card)
		}
	}
	rng := rand.New(rand.NewSource(seed))
	rng.Shuffle(len(pool), func(i int, j int) { pool[i], pool[j] = pool[j], pool[i] })
	forbidden := foundationVoids(spec)
	for seat := range hands {
		for len(hands[seat]) < len(spec.Hand) {
			index := -1
			for i, card := range pool {
				if !forbidden[seat][card.GetSuit()-1] {
					index = i
					break
				}
			}
			if index < 0 {
				return nil, fmt.Errorf("cannot complete legal hidden hands")
			}
			hands[seat] = append(hands[seat], pool[index])
			pool = append(pool[:index], pool[index+1:]...)
		}
	}
	return hands, nil
}
