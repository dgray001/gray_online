package game

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

type PlayerAction struct {
	action_id int
	Client_id int
	Kind      string
	Action    gin.H
	Ai_id     int
}

func (a *PlayerAction) toFrontend() gin.H {
	return gin.H{
		"action_id": a.action_id,
		"client_id": a.Client_id,
		"kind":      a.Kind,
		"action":    a.Action,
		"ai_id":     a.Ai_id,
	}
}

type Player struct {
	client_id        uint64
	ai_player_id     uint32 // specific to the game
	ai_running       bool   // can be true for human players
	Player_id        int
	nickname         string
	connected        bool
	Updates          chan *UpdateMessage
	FailedUpdates    chan *UpdateMessage
	AiUpdates        chan *UpdateMessage
	AiFailedUpdates  chan *UpdateMessage
	FlushConnections chan bool
	update_list      []*UpdateMessage
	base_game        *GameBase
}

const UPDATE_CHANNEL_SIZE = 2

func CreatePlayer(client_id uint64, nickname string, base_game *GameBase) {
	player := &Player{
		client_id:        client_id,
		ai_player_id:     0,
		ai_running:       false,
		Player_id:        -1,
		nickname:         nickname,
		connected:        false,
		Updates:          make(chan *UpdateMessage, UPDATE_CHANNEL_SIZE),
		AiUpdates:        make(chan *UpdateMessage, UPDATE_CHANNEL_SIZE),
		FailedUpdates:    make(chan *UpdateMessage, UPDATE_CHANNEL_SIZE),
		AiFailedUpdates:  make(chan *UpdateMessage, UPDATE_CHANNEL_SIZE),
		FlushConnections: make(chan bool, 2),
		update_list:      []*UpdateMessage{},
		base_game:        base_game,
	}
	base_game.Players[client_id] = player
}

func CreateAiPlayer(nickname string, base_game *GameBase) *Player {
	ai_id := base_game.NextAiId()
	player := &Player{
		client_id:        0,
		ai_player_id:     ai_id,
		ai_running:       true,
		Player_id:        -1,
		nickname:         nickname,
		connected:        false,
		Updates:          make(chan *UpdateMessage, UPDATE_CHANNEL_SIZE),
		AiUpdates:        make(chan *UpdateMessage, UPDATE_CHANNEL_SIZE),
		FailedUpdates:    make(chan *UpdateMessage, UPDATE_CHANNEL_SIZE),
		AiFailedUpdates:  make(chan *UpdateMessage, UPDATE_CHANNEL_SIZE),
		FlushConnections: make(chan bool, 2),
		update_list:      []*UpdateMessage{},
		base_game:        base_game,
	}
	base_game.AiPlayers[ai_id] = player
	return player
}

func (p *Player) IsHumanPlayer() bool {
	return p.client_id > 0
}

func (p *Player) RunningAi() {
	p.ai_running = true
}

func (p *Player) AddUpdate(update *UpdateMessage) {
	if !p.base_game.game_started {
		fmt.Fprintln(os.Stderr, "Can't add update to game that isn't started")
		return
	}
	if p.base_game.game_ended {
		fmt.Fprintln(os.Stderr, "Can't add update to game that is ended")
		return
	}
	if !p.base_game.PersistantHistory() {
		p.update_list = make([]*UpdateMessage, 0)
	}
	update.Id = len(p.update_list) + 1 // start at 1
	p.update_list = append(p.update_list, update)
	if p.IsHumanPlayer() {
		p.Updates <- update
	}
	if p.ai_running {
		p.AiUpdates <- update
	}
}

func (p *Player) AddFailedUpdate(update *UpdateMessage) {
	if p.IsHumanPlayer() {
		p.FailedUpdates <- update
	}
	if p.ai_running {
		p.AiFailedUpdates <- update
	}
}

func (p *Player) AddFailedUpdateShorthand(kind string, message string) {
	fmt.Println(message)
	p.AddFailedUpdate(&UpdateMessage{Kind: kind, Content: gin.H{"message": message, "player_id": p.Player_id}})
}

func (p *Player) GetNickname() string {
	return p.nickname
}

func (p *Player) GetConnected() bool {
	return p.connected
}

func (p *Player) GetClientId() uint64 {
	return p.client_id
}

func (p *Player) GetBase() *GameBase {
	return p.base_game
}

func (p *Player) GetAiId() uint32 {
	return p.ai_player_id
}

func (p *Player) ToFrontend(show_updates bool) gin.H {
	player := gin.H{
		"client_id":    p.client_id,
		"ai_player_id": p.ai_player_id,
		"player_id":    p.Player_id,
		"nickname":     p.nickname,
		"connected":    p.connected,
	}
	if show_updates {
		updates := []gin.H{}
		for _, update := range p.update_list {
			if update != nil {
				updates = append(updates, update.toFrontend())
			}
		}
		player["updates"] = updates
	}
	return player
}
