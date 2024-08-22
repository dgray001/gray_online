package test_game

import (
	"github.com/dgray001/gray_online/game"
	"github.com/gin-gonic/gin"
)

type TestGamePlayer struct {
	player *game.Player
}

func (p *TestGamePlayer) toFrontend(show_updates bool) gin.H {
	player := gin.H{}
	if p.player != nil {
		player["player"] = p.player.ToFrontend(show_updates)
	}
	return player
}
