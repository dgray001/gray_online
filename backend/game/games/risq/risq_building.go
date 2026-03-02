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
	turn_stamina       int
	current_stamina    int
	max_stamina        int
	cs                 RisqCombatStats
	active_orders      []*RisqOrder
	past_orders        []*RisqOrder
	intent             *RisqBuildingIntent
}

func createRisqBuilding(internal_id uint64, building_id uint32, player_id int) *RisqBuilding {
	building := RisqBuilding{
		deleted:            false,
		internal_id:        internal_id,
		player_id:          player_id,
		building_id:        building_id,
		population_support: 0,
		turn_stamina:       10,
		current_stamina:    0,
		max_stamina:        15,
		cs:                 createRisqCombatStats(),
		active_orders:      make([]*RisqOrder, 0),
		past_orders:        make([]*RisqOrder, 0),
		intent:             nil,
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

func (b *RisqBuilding) score() uint {
	return 0
}

func (b *RisqBuilding) isDeleted() bool {
	return b.deleted
}

func (b *RisqBuilding) internalId() uint64 {
	return b.internal_id
}

func (b *RisqBuilding) refreshStamina() {
	b.current_stamina += b.turn_stamina
	if b.current_stamina > b.max_stamina {
		b.current_stamina = b.max_stamina
	}
}

func (b *RisqBuilding) receiveOrder(o *RisqOrder) {
	// TODO: if order costs resources then remove resources here
	b.past_orders = append(b.past_orders, o)
	b.active_orders = append(b.active_orders, o)
}

func (b *RisqBuilding) orderComplete(o *RisqOrder, risq *GameRisq) bool {
	// TODO: implement
	return true
}

func (b *RisqBuilding) tickIntent(risq *GameRisq) bool {
	b.intent = nil
	if len(b.active_orders) == 0 {
		return false
	}
	// TODO: implement
	panic("BUILDING ORDERS NOT IMPLEMENTED!!")
	// return b.intent != nil
}

func (b *RisqBuilding) tickExecute(risq *GameRisq) {
	if b.intent == nil {
		return
	}
	// TODO: implement
	panic("BUILDING ORDERS NOT IMPLEMENTED!!")
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
	active_orders := make([]gin.H, 0)
	for _, order := range b.active_orders {
		if order != nil && !order.executed {
			active_orders = append(active_orders, order.toFrontend())
		}
	}
	building["active_orders"] = active_orders
	return building
}
