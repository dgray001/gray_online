package fiddlesticks

import (
	"maps"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/gin-gonic/gin"
)

const timeoutActionKey = "timeout"

type aiChoice struct {
	model_id uint8
	bet      uint8
	card     *game_utils.StandardCard
}

func isTimeoutAction(action game.PlayerAction) bool {
	timeout, _ := action.Action[timeoutActionKey].(bool)
	return timeout
}

func (p *FiddlesticksPlayer) betComparison(action game.PlayerAction, bet uint8) gin.H {
	choice := p.ai_choice
	p.ai_choice = nil
	if choice == nil || choice.card != nil || isTimeoutAction(action) {
		return nil
	}
	return gin.H{"model_id": choice.model_id, "ai_choice": choice.bet, "matched": choice.bet == bet}
}

func (p *FiddlesticksPlayer) cardComparison(action game.PlayerAction, card *game_utils.StandardCard) gin.H {
	choice := p.ai_choice
	p.ai_choice = nil
	if choice == nil || choice.card == nil || isTimeoutAction(action) {
		return nil
	}
	return gin.H{"model_id": choice.model_id, "ai_choice": choice.card.ToFrontend(), "matched": choice.card == card}
}

func (f *GameFiddlesticks) broadcastAction(actor *game.Player, update *game.UpdateMessage, comparison gin.H) {
	if comparison == nil {
		game.Game_BroadcastUpdate(f, update)
		return
	}
	own := &game.UpdateMessage{Kind: update.Kind, Content: maps.Clone(update.Content)}
	own.Content["ai_comparison"] = comparison
	base := f.GetBase()
	for _, player := range base.Players {
		if player == actor {
			player.AddUpdate(own)
		} else {
			player.AddUpdate(update)
		}
	}
	for _, player := range base.AiPlayers {
		player.AddUpdate(update)
	}
	base.AddViewerUpdate(update)
}
