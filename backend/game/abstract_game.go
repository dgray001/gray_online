package game

import (
	"github.com/gin-gonic/gin"
)

type GameBase struct {
	Game_id   uint64
	game_type uint8
	// here the map keys are the lobby client ids
	Players      map[uint64]*Player
	Viewers      map[uint64]*Viewer
	game_started bool
	game_ended   bool
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
	ClientId int
	Kind     string
	Action   gin.H
}

type Game interface {
	GetId() uint64
	GetBase() *GameBase
	StartGame()
	Valid() bool
	ToFrontend() gin.H
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

func (g *GameBase) gameEnded() bool {
	return g.game_ended
}

func (g *GameBase) EndGame() {
	if g.game_ended {
		panic("Game already ended")
	}
	g.game_ended = true
}

func (g *GameBase) ToFrontend() gin.H {
	game_base := gin.H{
		"game_id":      g.Game_id,
		"game_type":    g.game_type,
		"game_started": g.game_started,
		"game_ended":   g.game_ended,
	}
	players := []gin.H{}
	for _, player := range g.Players {
		if player != nil {
			players = append(players, player.ToFrontend())
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
	return game_base
}

type UpdateMessage struct {
	Kind    string
	Content gin.H
}

type Player struct {
	client_id uint64
	Player_id int
	nickname  string
	connected bool
	Updates   chan *UpdateMessage
}

func CreatePlayer(client_id uint64, nickname string) *Player {
	return &Player{
		client_id: client_id,
		Player_id: -1,
		nickname:  nickname,
		connected: false,
		Updates:   make(chan *UpdateMessage),
	}
}

func (p *Player) AddUpdate(update *UpdateMessage) {
	// TODO: track with integer to ensure no duplicates, etc... also store so client can request an update they missed
	p.Updates <- update
}

func (p *Player) ToFrontend() gin.H {
	return gin.H{
		"client_id": p.client_id,
		"player_id": p.Player_id,
		"nickname":  p.nickname,
		"connected": p.connected,
	}
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
