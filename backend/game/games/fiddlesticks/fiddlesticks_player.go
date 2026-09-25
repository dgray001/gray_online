package fiddlesticks

import (
	"fmt"
	"sync"
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

// clearTurnTimer assumes the caller already holds the game's mutex.
func (p *FiddlesticksPlayer) clearTurnTimer() {
	if p.turn_timer != nil {
		p.turn_timer.Stop()
		p.turn_timer = nil
	}
}

// storeTurnAction assumes the caller already holds mu; the fired timer re-acquires it to
// check p.turn_timer is still this timer, so a real action that beat the timer (and cleared it
// under the same lock) prevents the stale action from being sent.
func (p *FiddlesticksPlayer) storeTurnAction(action game.PlayerAction, action_channel chan game.PlayerAction, d time.Duration, mu *sync.Mutex) {
	p.turn_start_time = time.Now()
	var timer *time.Timer
	timer = time.AfterFunc(d, func() {
		mu.Lock()
		defer mu.Unlock()
		if p.turn_timer != timer {
			return
		}
		p.turn_timer = nil
		fmt.Println("Turn timer up so AI playing turn for", p.player.Player_id)
		action_channel <- action
	})
	p.turn_timer = timer
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
		player["turn_start_time"] = p.turn_start_time.UnixMilli()
	}
	cards := []gin.H{}
	if show_updates {
		for _, card := range p.cards {
			if card != nil {
				cards = append(cards, card.ToFrontend())
			}
		}
	}
	player["cards"] = cards
	return player
}
