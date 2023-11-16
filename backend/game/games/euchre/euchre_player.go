package euchre

import (
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/gin-gonic/gin"
)

type EuchrePlayer struct {
	player       *game.Player
	cards        []*game_utils.StandardCard
	cards_played []int
}

func (p *EuchrePlayer) toFrontend(show_updates bool) gin.H {
	player := gin.H{
		"cards_played": p.cards_played,
	}
	if p.player != nil {
		player["player"] = p.player.ToFrontend(show_updates)
	}
	cards := []gin.H{}
	for _, card := range p.cards {
		if card != nil {
			cards = append(cards, card.ToFrontend())
		}
	}
	player["cards"] = cards
	return player
}
