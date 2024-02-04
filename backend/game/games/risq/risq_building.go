package risq

import (
	"github.com/gin-gonic/gin"
)

type RisqBuilding struct {
	deleted            bool
	internal_id        uint64
	player_id          int
	building_id        uint32
	display_name       string
	zone               *RisqZone
	population_support uint16
	cs                 RisqCombatStats
}

func createRisqBuilding(internal_id uint64, building_id uint32, player_id int) *RisqBuilding {
	building := RisqBuilding{
		deleted:            false,
		internal_id:        internal_id,
		player_id:          player_id,
		building_id:        building_id,
		population_support: 0,
		cs:                 createRisqCombatStats(),
	}
	switch building_id {
	case 1: // village center
		building.display_name = "Village Center"
		building.cs.setMaxHealth(350)
		building.population_support = 5
	case 2: // housing
		building.display_name = "Housing"
		building.cs.setMaxHealth(120)
		building.population_support = 5
	}
	return &building
}

func (b *RisqBuilding) vision() *RisqVision {
	return &RisqVision{
		space:         4,
		edge_adjacent: 4,
		adjacent:      3,
		edge_opposite: 2,
		secondary:     0,
	}
}

func (b *RisqBuilding) toFrontend() gin.H {
	building := gin.H{
		"internal_id":        b.internal_id,
		"player_id":          b.player_id,
		"building_id":        b.building_id,
		"display_name":       b.display_name,
		"population_support": b.population_support,
		"combat_stats":       b.cs.toFrontend(),
	}
	if b.zone != nil {
		building["zone_coordinate"] = b.zone.coordinate.ToFrontend()
		if b.zone.space != nil {
			building["space_coordinate"] = b.zone.space.coordinate.ToFrontend()
		}
	}
	return building
}
