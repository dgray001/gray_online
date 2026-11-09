package fiddlesticks

import (
	"fmt"
	"github.com/dgray001/gray_online/game"
)

func (f *GameFiddlesticks) ApplyLearningAction(action int) error {
	if !f.game.GameStarted() || f.game.GameEnded() {
		return fmt.Errorf("game is not accepting actions")
	}
	p := f.players[f.turn]
	updates := []*game.UpdateMessage{}
	if f.betting {
		if action < 0 || action > int(f.round) {
			return fmt.Errorf("invalid bid")
		}
		updates = append(updates, f.executeBet(p.player, uint8(action), false, nil))
	} else {
		valid, _ := f.validCards(f.turn)
		index := -1
		for _, i := range valid {
			card := p.cards[i]
			if 27+int(card.GetSuit()-1)*13+int(card.GetNumber()-2) == action {
				index = i
				break
			}
		}
		if index < 0 {
			return fmt.Errorf("card is not legal")
		}
		updates = f.executePlayCard(p.player, index, false, nil)
	}
	for _, update := range updates {
		for _, player := range f.players {
			player.ai_model.ApplyUpdate(player, f, update)
		}
	}
	return nil
}
