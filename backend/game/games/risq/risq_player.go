package risq

import (
	"github.com/dgray001/gray_online/game"
	"github.com/gin-gonic/gin"
)

type RisqPlayer struct {
	player    *game.Player
	resources *RisqPlayerResources
	buildings []*RisqBuilding
	units     []*RisqUnit
}

func createRisqPlayer(player *game.Player) *RisqPlayer {
	return &RisqPlayer{
		player:    player,
		resources: createRisqPlayerResources(),
		buildings: make([]*RisqBuilding, 0),
		units:     make([]*RisqUnit, 0),
	}
}

func (p *RisqPlayer) toFrontend(show_updates bool) gin.H {
	player := gin.H{}
	if p.player != nil {
		player["player"] = p.player.ToFrontend(show_updates)
	}
	if p.resources != nil {
		player["resources"] = p.resources.toFrontend()
	}
	buildings := []gin.H{}
	for _, building := range p.buildings {
		if building != nil && !building.deleted {
			buildings = append(buildings, building.toFrontend())
		}
	}
	player["buildings"] = buildings
	return player
}
