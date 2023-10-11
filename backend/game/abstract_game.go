package game

import (
	"github.com/gin-gonic/gin"
)

type GameBase struct {
	Game_id   uint64
	game_type uint8
	// here the map keys are the lobby client ids
	Players        map[uint64]*Player
	Viewers        map[uint64]*Viewer
	game_started   bool
	game_ended     bool
	player_updates []*PlayerAction
}

func CreateBaseGame(game_id uint64, game_type uint8) *GameBase {
	return &GameBase{
		Game_id:      game_id,
		game_type:    game_type,
		Players:      make(map[uint64]*Player),
		Viewers:      make(map[uint64]*Viewer),
		game_started: false,
		game_ended:   false,
	}
}

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

type Game interface {
	GetId() uint64
	GetBase() *GameBase
	StartGame()
	Valid() bool
	ToFrontend(client_id uint64, viewer bool) gin.H
	PlayerAction(action PlayerAction)
}

func (g *GameBase) PlayerConnected(client_id uint64) bool {
	player := g.Players[client_id]
	if player == nil || player.connected {
		return false
	}
	player.connected = true
	for _, player := range g.Players {
		if !player.connected {
			return false
		}
	}
	return true
}

func (g *GameBase) GameStarted() bool {
	return g.game_started
}

func (g *GameBase) StartGame() {
	if g.game_started {
		panic("Game already started")
	}
	g.game_started = true
}

func (g *GameBase) AddAction(action PlayerAction) PlayerAction {
	action.action_id = len(g.player_updates) + 1 // start at 1
	g.player_updates = append(g.player_updates, &action)
	return action
}

func (g *GameBase) ResendPlayerUpdate(client_id uint64, update_id int) {
	player := g.Players[client_id]
	if player == nil || update_id < 1 || update_id > len(player.update_list) {
		return
	}
	player.Updates <- player.update_list[update_id-1]
}

func (g *GameBase) ResendLastUpdate(client_id uint64) {
	player := g.Players[client_id]
	if player == nil || len(player.update_list) < 1 {
		return
	}
	player.Updates <- player.update_list[len(player.update_list)-1]
}

func (g *GameBase) gameEnded() bool {
	return g.game_ended
}

func (g *GameBase) EndGame() {
	if g.game_ended {
		panic("Game already ended")
	}
	g.game_ended = true
}

func (g *GameBase) ToFrontend(client_id uint64, is_viewer bool) gin.H {
	game_base := gin.H{
		"game_id":      g.Game_id,
		"game_type":    g.game_type,
		"game_started": g.game_started,
		"game_ended":   g.game_ended,
	}
	players := []gin.H{}
	for _, player := range g.Players {
		if player != nil {
			players = append(players, player.ToFrontend(is_viewer || client_id == player.client_id))
		}
	}
	game_base["players"] = players
	viewers := []gin.H{}
	for _, viewer := range g.Viewers {
		if viewer != nil {
			viewers = append(viewers, viewer.ToFrontend())
		}
	}
	game_base["viewers"] = viewers
	if is_viewer {
		player_actions := []gin.H{}
		for _, player_action := range g.player_updates {
			if player_action != nil {
				player_actions = append(player_actions, player_action.toFrontend())
			}
		}
		game_base["player_actions"] = player_actions
	}
	return game_base
}

type UpdateMessage struct {
	Id      int
	Kind    string
	Content gin.H
}

func (u *UpdateMessage) toFrontend() gin.H {
	return gin.H{
		"id":      u.Id,
		"kind":    u.Kind,
		"content": u.Content,
	}
}

type Player struct {
	client_id   uint64
	Player_id   int
	nickname    string
	connected   bool
	Updates     chan *UpdateMessage
	update_list []*UpdateMessage
}

func CreatePlayer(client_id uint64, nickname string) *Player {
	return &Player{
		client_id:   client_id,
		Player_id:   -1,
		nickname:    nickname,
		connected:   false,
		Updates:     make(chan *UpdateMessage),
		update_list: []*UpdateMessage{},
	}
}

func (p *Player) AddUpdate(update *UpdateMessage) {
	update.Id = len(p.update_list) + 1 // start at 1
	p.update_list = append(p.update_list, update)
	p.Updates <- update
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

type Viewer struct {
	client_id uint64
	nickname  string
	connected bool
}

func CreateViewer(client_id uint64, nickname string) *Viewer {
	return &Viewer{
		client_id: client_id,
		nickname:  nickname,
		connected: false,
	}
}

func (v *Viewer) ToFrontend() gin.H {
	return gin.H{
		"client_id": v.client_id,
		"nickname":  v.nickname,
		"connected": v.connected,
	}
}
