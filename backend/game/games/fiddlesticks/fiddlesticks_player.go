package fiddlesticks

import (
	"fmt"
	"time"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/gin-gonic/gin"
)

type FiddlesticksPlayer struct {
	player                *game.Player
	cards                 []*game_utils.StandardCard
	cards_played          []int
	score                 uint32
	bet                   uint8
	has_bet               bool
	tricks                uint8
	ai_model_id           uint8
	ai_model              FiddlesticksAiModel
	instantiated_ai_model bool
	turn_timer            *time.Timer
	turn_start_time       time.Time
}

func (p *FiddlesticksPlayer) createAiModel(model_input map[string]string) {
	p.instantiated_ai_model = true
	switch p.ai_model_id {
	case 0: // random model
		p.ai_model = &FiddlesticksAiModelRandom{}
	case 1: // theory model 1
		p.ai_model = createFiddlesticksAiModelTheory1(model_input)
	case 2: // theory model 2
		p.ai_model = createFiddlesticksAiModelTheory2(model_input)
	case 3: // theory model 3
		p.ai_model = createFiddlesticksAiModelTheory3(model_input)
	case 4: // theory model 4
		p.ai_model = createFiddlesticksAiModelTheory4(model_input)
	default:
		p.ai_model = &FiddlesticksAiModelRandom{}
	}
}

func (p *FiddlesticksPlayer) instantiatedAiModel() bool {
	return p.instantiated_ai_model
}

func (p *FiddlesticksPlayer) clearTurnTimer() {
	if p.turn_timer != nil {
		p.turn_timer.Stop()
		p.turn_timer = nil
	}
}

func (p *FiddlesticksPlayer) storeTurnAction(action game.PlayerAction, action_channel chan game.PlayerAction, d time.Duration) {
	turn_timer := time.NewTimer(d)
	p.turn_timer = turn_timer
	go func() {
		<-turn_timer.C
		fmt.Println("Turn timer up so AI playing turn for", p.player.Player_id)
		action_channel <- action
	}()
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
	if p.turn_timer != nil {
		player["elapsed_time"] = time.Since(p.turn_start_time).Milliseconds()
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
