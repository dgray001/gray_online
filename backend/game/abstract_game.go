package game

import (
	"fmt"
	"os"

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
	} else {
		fmt.Fprintln(os.Stderr, "Game base is nil")
	}
	return 0
}

func Game_StartGame(g Game) {
	base := g.GetBase()
	if base != nil {
		base.StartGame()
	} else {
		fmt.Fprintln(os.Stderr, "Game base is nil")
	}
	fmt.Println("Starting game id ", base.Game_id)
	g.StartGame()
}

func Game_BroadcastUpdate(g Game, update *UpdateMessage) {
	base := g.GetBase()
	if base != nil {
		fmt.Printf("Broadcasting game (%d) update {%s, %s}\n", base.Game_id, update.Kind, update.Content)
		for _, player := range base.Players {
			player.AddUpdate(update)
		}
		for _, player := range base.AiPlayers {
			player.AddUpdate(update)
		}
		base.AddViewerUpdate(update)
	} else {
		fmt.Fprintln(os.Stderr, "Game base is nil")
	}
}

func Game_BroadcastAiUpdate(g Game, update *UpdateMessage) {
	base := g.GetBase()
	if base != nil {
		for _, player := range base.AiPlayers {
			//player.Updates <- update
			player.AddUpdate(update)
		}
	} else {
		fmt.Fprintln(os.Stderr, "Game base is nil")
	}
}

// Returns whether this is the last player to disconnect
func Game_PlayerDisconnected(g Game, client_id uint64) bool {
	g.PlayerDisconnected(client_id)
	base := g.GetBase()
	if base != nil {
		return base.PlayerDisconnected(client_id)
	} else {
		fmt.Fprintln(os.Stderr, "Game base is nil")
	}
	return false
}

// Returns whether player really reconnected
func Game_PlayerReconnected(g Game, client_id uint64) bool {
	base := g.GetBase()
	if base == nil {
		fmt.Fprintln(os.Stderr, "Game base is nil")
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
