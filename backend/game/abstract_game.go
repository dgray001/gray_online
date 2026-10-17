package game

import (
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

type Game interface {
	GetBase() *GameBase
	StartGame()
	Valid() bool
	PlayerAction(action PlayerAction)
	PlayerDisconnected(client_id uint64)
	PlayerReconnected(client_id uint64)
	ToFrontend(client_id uint64, viewer bool) gin.H
}

func Game_GetId(g Game) uint64 {
	base := g.GetBase()
	if base != nil {
		return base.Game_id
	}
	util.Error(util.DefaultErr, "Game base is nil")
	return 0
}

func Game_StartGame(g Game) {
	base := g.GetBase()
	if base != nil {
		util.Infof(base.Std, "Starting game id %d", base.Game_id)
		base.StartGame()
		g.StartGame() // must be after base.StartGame() in case this sends game updates
	} else {
		util.Error(util.DefaultErr, "Game base is nil")
	}
}

func Game_BroadcastUpdate(g Game, update *UpdateMessage) {
	base := g.GetBase()
	if base != nil {
		util.Debugf(base.Std, "Broadcasting game (%d) update {%s, %s}", base.Game_id, update.Kind, update.Content)
		for _, player := range base.Players {
			player.AddUpdate(update)
		}
		for _, player := range base.AiPlayers {
			player.AddUpdate(update)
		}
		base.AddViewerUpdate(update)
	} else {
		util.Error(util.DefaultErr, "Game base is nil")
	}
}

// Returns whether this is the last player to disconnect
func Game_PlayerDisconnected(g Game, client_id uint64) bool {
	g.PlayerDisconnected(client_id)
	base := g.GetBase()
	if base != nil {
		return base.PlayerDisconnected(client_id)
	} else {
		util.Error(util.DefaultErr, "Game base is nil")
	}
	return false
}

// Returns whether player really reconnected
func Game_PlayerReconnected(g Game, client_id uint64) bool {
	base := g.GetBase()
	if base == nil {
		util.Error(util.DefaultErr, "Game base is nil")
		return false
	}
	g.PlayerReconnected(client_id)
	player := base.Players[client_id]
	if player != nil && !player.connected {
		base.PlayerConnected(client_id)
		return true
	}
	return false
}
