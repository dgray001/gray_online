package game

import (
	"fmt"
	"os"
	"sync"

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
	update_list_mu   sync.RWMutex
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
	p.update_list_mu.Lock()
	if !p.base_game.PersistantHistory() {
		p.update_list = make([]*UpdateMessage, 0)
	}
	// copy so this recipient's Id assignment can't race other recipients' AddUpdate/AddViewerUpdate calls on the shared update
	own_update := *update
	own_update.Id = len(p.update_list) + 1 // start at 1
	p.update_list = append(p.update_list, &own_update)
	p.update_list_mu.Unlock()
	if p.IsHumanPlayer() {
		select {
		case p.Updates <- &own_update:
		default:
			fmt.Fprintln(os.Stderr, "Dropped update to player", p.Player_id, "- update buffer full:", own_update.Kind)
		}
	}
	if p.ai_running {
		select {
		case p.AiUpdates <- &own_update:
		default:
			fmt.Fprintln(os.Stderr, "Dropped update to AI player", p.Player_id, "- update buffer full:", own_update.Kind)
		}
	}
}

// ReplayUpdates calls apply, in order, for every update after last_applied_id in this player's
// permanent update history, returning the new last_applied_id. This is for games whose AI
// decision loop wakes up on an AiUpdates channel receive but shouldn't trust that channel to
// deliver every update exactly once (AddUpdate drops updates non-blocking when that channel's
// buffer is full) — walking the permanent history instead makes a dropped wakeup harmless, the
// same way the frontend replays from its own update history by id. Only meaningful for games with
// GameBase.PersistantHistory() true; a non-persistent game's update_list is reset on every update.
func (p *Player) ReplayUpdates(last_applied_id int, apply func(*UpdateMessage)) int {
	p.update_list_mu.RLock()
	defer p.update_list_mu.RUnlock()
	for last_applied_id < len(p.update_list) {
		apply(p.update_list[last_applied_id])
		last_applied_id++
	}
	return last_applied_id
}

func (p *Player) AddFailedUpdate(update *UpdateMessage) {
	if p.IsHumanPlayer() {
		select {
		case p.FailedUpdates <- update:
		default:
			fmt.Fprintln(os.Stderr, "Dropped failed update to player", p.Player_id, "- buffer full:", update.Kind)
		}
	}
	if p.ai_running {
		select {
		case p.AiFailedUpdates <- update:
		default:
			fmt.Fprintln(os.Stderr, "Dropped failed update to AI player", p.Player_id, "- buffer full:", update.Kind)
		}
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
