package fiddlesticks

import (
	"fmt"
	"math"
	"math/rand"
	"os"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

func runAi(p *FiddlesticksPlayer, f *GameFiddlesticks, action_channel chan game.PlayerAction) {
	fmt.Println("Starting ai for AI player", p.player.GetAiId())
	for {
		if p == nil || p.player == nil || p.player.GetBase() == nil || p.player.GetBase().GameEnded() {
			break
		}
		if f == nil || f.GetBase() == nil {
			break
		}
		if !p.instantiatedAiModel() {
			fmt.Println("Instantiating model for AI player", p.player.GetAiId())
			p.createAiModel(nil)
		}
		select {
		case update := <-p.player.AiUpdates:
			fmt.Println("AI player", p.player.GetAiId(), "received update", update)
			if p.player == nil || p.player.GetBase() == nil || !p.player.GetBase().GameStarted() || p.player.GetBase().GameEnded() {
				break
			}
			if f.GetBase() == nil || !f.GetBase().GameStarted() || f.GetBase().GameEnded() {
				break
			}
			p.ai_model.ApplyUpdate(p, f, update)
			checkTurn(p, f, action_channel)
			fmt.Println("Finished checking for AI player", p.player.GetAiId())
		case update := <-p.player.FailedUpdates:
			fmt.Fprintln(os.Stderr, "AI player", p.player.GetAiId(), "received failed update", update)
		}
	}
	fmt.Println("Ending ai for AI player", p.player.GetAiId())
}

func checkTurn(p *FiddlesticksPlayer, f *GameFiddlesticks, action_channel chan game.PlayerAction) {
	p.clearTurnTimer()
	if f.turn != p.player.Player_id {
		return
	}
	if f.betting {
		bid := GetAiBid(p, f)
		action := gin.H{
			"amount": float64(bid),
		}
		player_action := createPlayerAction(p.player, "bet", action)
		if p.player.IsHumanPlayer() {
			fmt.Println("Storing bet for human player", p.player.Player_id, ":", bid)
			p.storeTurnAction(player_action, action_channel, f.turn_duration)
			return
		}
		fmt.Println("Betting for ai player", p.player.Player_id, ":", bid)
		action_channel <- player_action
	} else {
		card_index := GetAiPlayCard(p, f)
		action := gin.H{
			"index": float64(card_index),
		}
		player_action := createPlayerAction(p.player, "play-card", action)
		if p.player.IsHumanPlayer() {
			fmt.Println("Storing play card for human player", p.player.Player_id, ":", p.cards[card_index].GetName())
			p.storeTurnAction(player_action, action_channel, f.turn_duration)
			return
		}
		fmt.Println("Playing card for ai player", p.player.Player_id, ":", p.cards[card_index].GetName())
		action_channel <- player_action
	}
}

func createPlayerAction(p *game.Player, kind string, action gin.H) game.PlayerAction {
	player_action := game.PlayerAction{Kind: kind, Action: action}
	if p.GetClientId() > 0 {
		player_action.Client_id = int(p.GetClientId())
	} else {
		player_action.Ai_id = int(p.GetAiId())
	}
	return player_action
}

func GetAiBid(p *FiddlesticksPlayer, f *GameFiddlesticks) uint8 {
	raw_bid := p.ai_model.Bet(p, f)
	bid := uint8(raw_bid)
	if util.RandomChance(math.Mod(raw_bid, 1)) {
		bid += 1
	}
	return bid
}

func GetAiPlayCard(p *FiddlesticksPlayer, f *GameFiddlesticks) int {
	valid_cards, _ := f.validCards(p.player.Player_id)
	if len(valid_cards) == 0 {
		fmt.Fprintln(os.Stderr, "No valid cards for AI to play")
		return 0
	}
	card_weights := p.ai_model.CardWeights(p, f, valid_cards)
	total_weight := float64(0)
	for _, weight := range card_weights {
		total_weight += weight
	}
	card_index := 0
	random_choice := total_weight * rand.Float64()
	for i, weight := range card_weights {
		if random_choice <= float64(weight) {
			card_index = i
			break
		}
		random_choice -= float64(weight)
	}
	return valid_cards[card_index]
}

type FiddlesticksAiModel interface {
	// applies an update, allowing for the update of information
	ApplyUpdate(p *FiddlesticksPlayer, f *GameFiddlesticks, update *game.UpdateMessage)
	// returns a float value which is then resolved into an int based on probability
	Bet(p *FiddlesticksPlayer, f *GameFiddlesticks) float64
	// returns un-normalized weights which must be the same length as the valid_Cards slice
	CardWeights(p *FiddlesticksPlayer, f *GameFiddlesticks, valid_cards []int) []float64
}
