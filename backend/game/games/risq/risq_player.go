package risq

import (
	"github.com/dgray001/gray_online/game"
	"github.com/gin-gonic/gin"
)

type RisqPlayer struct {
	player    *game.Player
	resources *RisqPlayerResources
	buildings map[uint64]*RisqBuilding
	units     map[uint64]*RisqUnit
}

func createRisqPlayer(player *game.Player) *RisqPlayer {
	return &RisqPlayer{
		player:    player,
		resources: createRisqPlayerResources(),
		buildings: make(map[uint64]*RisqBuilding),
		units:     make(map[uint64]*RisqUnit, 0),
	}
}

func (p *RisqPlayer) populationLimit() uint16 {
	limit := uint16(0)
	for _, building := range p.buildings {
		if building != nil && !building.deleted {
			limit += building.population_support
		}
	}
	return limit
}

func (p *RisqPlayer) toFrontend(show_updates bool) gin.H {
	player := gin.H{
		"population_limit": p.populationLimit(),
	}
	if p.player != nil {
		player["player"] = p.player.ToFrontend(show_updates)
	}
	if p.resources != nil {
		player["resources"] = p.resources.toFrontend()
	}
	buildings := make([]gin.H, 0)
	for _, building := range p.buildings {
		if building != nil && !building.deleted {
			buildings = append(buildings, building.toFrontend())
		}
	}
	player["buildings"] = buildings
	units := make([]gin.H, 0)
	for _, unit := range p.units {
		if unit != nil && !unit.deleted {
			units = append(units, unit.toFrontend())
		}
	}
	player["units"] = units
	return player
}
