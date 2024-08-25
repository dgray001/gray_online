package fiddlesticks

import (
	"fmt"
	"os"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

func runAi(p *game.Player, f *GameFiddlesticks, action_channel chan game.PlayerAction) {
	for {
		if p == nil || p.GetBase() == nil || p.GetBase().GameEnded() {
			break
		}
		select {
		case update := <-p.Updates:
			fmt.Println("AI player", p.GetAiId(), "received update", update)
			checkTurn(p, f, action_channel)
			fmt.Println("Finished checking for AI player", p.GetAiId())
		case update := <-p.FailedUpdates:
			fmt.Fprintln(os.Stderr, "AI player", p.GetAiId(), "received failed update", update)
		}
	}
}

func checkTurn(p *game.Player, f *GameFiddlesticks, action_channel chan game.PlayerAction) {
	if !f.GetBase().GameStarted() && f.GetBase().GameEnded() {
		return
	}
	if f.turn != p.Player_id {
		return
	}
	if f.betting {
		action := gin.H{
			"amount": 1.0,
		}
		player_action := game.PlayerAction{Kind: "bet", Ai_id: int(p.GetAiId()), Action: action}
		action_channel <- player_action
	} else {
		valid_cards, _ := f.validCards(p.Player_id)
		if len(valid_cards) == 0 {
			return
		}
		card_index := util.RandomInt(0, len(valid_cards)-1)
		action := gin.H{
			"index": float64(valid_cards[card_index]),
		}
		player_action := game.PlayerAction{Kind: "play-card", Ai_id: int(p.GetAiId()), Action: action}
		action_channel <- player_action
	}
}
