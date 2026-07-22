package egyptian_rat_crap

import (
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/gin-gonic/gin"
)

type EgyptianRatCrapPlayer struct {
	player  *game.Player
	pile    []*game_utils.StandardCard
	slapped bool
}

func (p *EgyptianRatCrapPlayer) toFrontend(show_updates bool) gin.H {
	player := gin.H{
		"pile_count": len(p.pile),
		"slapped":    p.slapped,
	}
	if p.player != nil {
		player["player"] = p.player.ToFrontend(show_updates)
	}
	return player
}
