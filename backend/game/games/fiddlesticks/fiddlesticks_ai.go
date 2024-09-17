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
			p.ai_model = p.createAiModel()
		}
		select {
		case update := <-p.player.Updates:
			fmt.Println("AI player", p.player.GetAiId(), "received update", update)
			checkTurn(p, f, action_channel)
			fmt.Println("Finished checking for AI player", p.player.GetAiId())
		case update := <-p.player.FailedUpdates:
			fmt.Fprintln(os.Stderr, "AI player", p.player.GetAiId(), "received failed update", update)
		}
	}
	fmt.Println("Ending ai for AI player", p.player.GetAiId())
}

func checkTurn(p *FiddlesticksPlayer, f *GameFiddlesticks, action_channel chan game.PlayerAction) {
	if p == nil || p.player == nil || p.player.GetBase() == nil || !p.player.GetBase().GameStarted() || p.player.GetBase().GameEnded() {
		return
	}
	if f == nil || f.GetBase() == nil || !f.GetBase().GameStarted() || f.GetBase().GameEnded() {
		return
	}
	if f.turn != p.player.Player_id {
		return
	}
	if f.betting {
		raw_bid := p.ai_model.Bet(p, f)
		bid := int(raw_bid)
		if util.RandomChance(math.Mod(raw_bid, 1)) {
			bid += 1
		}
		action := gin.H{
			"amount": float64(bid),
		}
		player_action := game.PlayerAction{Kind: "bet", Ai_id: int(p.player.GetAiId()), Action: action}
		action_channel <- player_action
	} else {
		valid_cards, _ := f.validCards(p.player.Player_id)
		if len(valid_cards) == 0 {
			return
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
		action := gin.H{
			"index": float64(valid_cards[card_index]),
		}
		player_action := game.PlayerAction{Kind: "play-card", Ai_id: int(p.player.GetAiId()), Action: action}
		action_channel <- player_action
	}
}

type FiddlesticksAiModel interface {
	// returns a float value which is then resolved into an int based on probability
	Bet(p *FiddlesticksPlayer, f *GameFiddlesticks) float64
	// returns un-normalized weights which must be the same length as the valid_Cards slice
	CardWeights(p *FiddlesticksPlayer, f *GameFiddlesticks, valid_cards []int) []float64
}
