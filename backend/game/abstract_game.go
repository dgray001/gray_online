package game

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

type GameBase struct {
	Game_id uint64
	// here the map keys are the lobby client ids
	Players      map[uint64]*Player
	Viewers      map[uint64]*Viewer
	game_started bool
	game_ended   bool
}

func CreateBaseGame(game_id uint64) *GameBase {
	return &GameBase{
		Game_id:      game_id,
		Players:      make(map[uint64]*Player),
		Viewers:      make(map[uint64]*Viewer),
		game_started: false,
		game_ended:   false,
	}
}

type Game interface {
	GetId() uint64
	StartGame()
	Valid() bool
	ToFrontend() gin.H
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
		"game_id":      strconv.Itoa(int(g.Game_id)),
		"game_started": strconv.FormatBool(g.game_started),
		"game_ended":   strconv.FormatBool(g.game_ended),
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

type Player struct {
	client_id uint64
}

func (p *Player) ToFrontend() gin.H {
	return gin.H{
		"client_id": p.client_id,
	}
}

type Viewer struct {
	client_id uint64
}

func (v *Viewer) ToFrontend() gin.H {
	return gin.H{
		"client_id": v.client_id,
	}
}
