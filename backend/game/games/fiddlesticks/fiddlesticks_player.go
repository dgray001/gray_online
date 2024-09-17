package fiddlesticks

import (
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/gin-gonic/gin"
)

type FiddlesticksPlayer struct {
	player                *game.Player
	cards                 []*game_utils.StandardCard
	cards_played          []int
	score                 uint16
	bet                   uint8
	has_bet               bool
	tricks                uint8
	ai_model_id           uint8
	ai_model              FiddlesticksAiModel
	instantiated_ai_model bool
}

func (p *FiddlesticksPlayer) createAiModel() FiddlesticksAiModel {
	p.instantiated_ai_model = true
	switch p.ai_model_id {
	case 0: // random model
		return FiddlesticksAiModelRandom{}
	case 1: // theory model 1
		return createFiddlesticksAiModelTheory1()
	default:
		return FiddlesticksAiModelRandom{}
	}
}

func (p *FiddlesticksPlayer) instantiatedAiModel() bool {
	return p.instantiated_ai_model
}

func (p *FiddlesticksPlayer) toFrontend(show_updates bool) gin.H {
	player := gin.H{
		"score":        p.score,
		"bet":          p.bet,
		"tricks":       p.tricks,
		"cards_played": p.cards_played,
		"has_bet":      p.has_bet,
		"ai_model_id":  p.ai_model_id,
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
