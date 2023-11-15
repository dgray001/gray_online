package game

import (
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
