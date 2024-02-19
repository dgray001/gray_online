package risq

import (
	"github.com/dgray001/gray_online/game"
	"github.com/gin-gonic/gin"
)

type RisqPlayer struct {
	player               *game.Player
	resources            *RisqPlayerResources
	buildings            map[uint64]*RisqBuilding
	units                map[uint64]*RisqUnit
	max_population_limit uint16
	color                string
}

func createRisqPlayer(player *game.Player, max_population_limit uint16, color string) *RisqPlayer {
	return &RisqPlayer{
		player:               player,
		resources:            createRisqPlayerResources(),
		buildings:            make(map[uint64]*RisqBuilding),
		units:                make(map[uint64]*RisqUnit, 0),
		max_population_limit: max_population_limit,
		color:                color,
	}
}

func (p *RisqPlayer) populationLimit() uint16 {
	limit := uint16(0)
	for _, building := range p.buildings {
		if building != nil && !building.deleted {
			limit += building.population_support
		}
	}
	if limit > p.max_population_limit {
		limit = p.max_population_limit
	}
	return limit
}

func (p *RisqPlayer) score() uint {
	score := uint(0)
	score += p.resources.score()
	for _, building := range p.buildings {
		score += building.score()
	}
	for _, unit := range p.units {
		score += unit.score()
	}
	return score
}

func (p *RisqPlayer) valid() bool {
	return true
}

func (p *RisqPlayer) toFrontend(show_updates bool) gin.H {
	player := gin.H{
		"population_limit": p.populationLimit(),
		"score":            p.score(),
		"color":            p.color,
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
