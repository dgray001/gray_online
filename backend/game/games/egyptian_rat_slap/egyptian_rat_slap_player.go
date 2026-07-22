package egyptian_rat_slap

import (
	"github.com/dgray001/gray_online/game"
	"github.com/gin-gonic/gin"
)

type EgyptianRatSlapPlayer struct {
	player *game.Player
}

func (p *EgyptianRatSlapPlayer) toFrontend(show_updates bool) gin.H {
	player := gin.H{}
	if p.player != nil {
		player["player"] = p.player.ToFrontend(show_updates)
	}
	return player
}
