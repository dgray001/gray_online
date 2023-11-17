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
}

func (a *PlayerAction) toFrontend() gin.H {
	return gin.H{
		"action_id": a.action_id,
		"client_id": a.Client_id,
		"kind":      a.Kind,
		"action":    a.Action,
	}
}

type UpdateMessage struct {
	Id      int
	Kind    string
	Content gin.H
}

func (u *UpdateMessage) toFrontend() gin.H {
	return gin.H{
		"update_id": u.Id,
		"kind":      u.Kind,
		"content":   u.Content,
	}
}

type Player struct {
	client_id     uint64
	Player_id     int
	nickname      string
	connected     bool
	Updates       chan *UpdateMessage
	FailedUpdates chan *UpdateMessage
	update_list   []*UpdateMessage
	base_game     *GameBase
}

func CreatePlayer(client_id uint64, nickname string, base_game *GameBase) *Player {
	return &Player{
		client_id:     client_id,
		Player_id:     -1,
		nickname:      nickname,
		connected:     false,
		Updates:       make(chan *UpdateMessage),
		FailedUpdates: make(chan *UpdateMessage),
		update_list:   []*UpdateMessage{},
		base_game:     base_game,
	}
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
	update.Id = len(p.update_list) + 1 // start at 1
	p.update_list = append(p.update_list, update)
	p.Updates <- update
}

func (p *Player) AddFailedUpdate(update *UpdateMessage) {
	p.FailedUpdates <- update
}

func (p *Player) AddFailedUpdateShorthand(kind string, message string) {
	fmt.Println(message)
	p.AddFailedUpdate(&UpdateMessage{Kind: kind, Content: gin.H{"message": message, "player_id": p.Player_id}})
}

func (p *Player) GetNickname() string {
	return p.nickname
}

func (p *Player) GetClientId() uint64 {
	return p.client_id
}

func (p *Player) ToFrontend(show_updates bool) gin.H {
	player := gin.H{
		"client_id": p.client_id,
		"player_id": p.Player_id,
		"nickname":  p.nickname,
		"connected": p.connected,
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
