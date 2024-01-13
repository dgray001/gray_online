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
	g.StartGame()
}

func Game_BroadcastUpdate(g Game, update *UpdateMessage) {
	base := g.GetBase()
	if base != nil {
		fmt.Printf("Broadcasting game (%d) update {%s, %s}\n", base.Game_id, update.Kind, update.Content)
		for _, player := range base.Players {
			player.AddUpdate(update)
		}
		base.AddViewerUpdate(update)
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

func Game_PlayerReconnected(g Game, client_id uint64) {
	g.PlayerReconnected(client_id)
	base := g.GetBase()
	if base != nil {
		base.PlayerConnected(client_id)
	} else {
		fmt.Fprintln(os.Stderr, "Game base is nil")
	}
}
