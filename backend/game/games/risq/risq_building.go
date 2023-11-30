package risq

import "github.com/gin-gonic/gin"

type RisqBuilding struct {
	deleted     bool
	internal_id uint64
	player_id   int
	building_id uint32
}

func createRisqBuilding(internal_id uint64, building_id uint32, player_id int) *RisqBuilding {
	building := RisqBuilding{
		deleted:     false,
		internal_id: internal_id,
		player_id:   player_id,
		building_id: building_id,
	}
	return &building
}

func (b *RisqBuilding) toFrontend() gin.H {
	building := gin.H{
		"internal_id": b.internal_id,
		"player_id":   b.player_id,
		"building_id": b.building_id,
	}
	return building
}
