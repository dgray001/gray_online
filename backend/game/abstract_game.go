package game

import (
	"fmt"

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
	return g.GetBase().Game_id
}

func Game_StartGame(g Game) {
	g.GetBase().StartGame()
	g.StartGame()
}

func Game_BroadcastUpdate(g Game, update *UpdateMessage) {
	fmt.Printf("Broadcasting game (%d) update {%s, %s}\n", g.GetBase().Game_id, update.Kind, update.Content)
	for _, player := range g.GetBase().Players {
		player.AddUpdate(update)
	}
	g.GetBase().AddViewerUpdate(update)
}
