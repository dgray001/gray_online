package risq

import (
	"fmt"
	"os"

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
	order_queue        RisqOrderQueue
	production_queue   []*RisqBuildingProductionItem
	intent             *RisqIntent
	// build stamina still needed to finish a unit-constructed foundation; 0 means not under construction
	stamina_remaining int
}

func (b *RisqBuilding) underConstruction() bool {
	return b.stamina_remaining > 0
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
		order_queue:        createRisqOrderQueue(),
		production_queue:   make([]*RisqBuildingProductionItem, 0),
		intent:             createRisqIntent(),
	}
	config, ok := buildingConfigs[building_id]
	if !ok {
		fmt.Fprintln(os.Stderr, "Creating unknown building id: ", building_id)
		return &building
	}
	building.display_name = config.display_name
	building.cs.setMaxHealth(config.max_health)
	building.population_support = config.population_support
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

type RisqBuildingProductionItem struct {
	// the order that queued this item, used to match it back up in orderComplete
	order             *RisqOrder
	unit_id           uint32
	stamina_remaining int
}

func (item *RisqBuildingProductionItem) toFrontend() gin.H {
	return gin.H{
		"unit_id":           item.unit_id,
		"stamina_remaining": item.stamina_remaining,
	}
}

func (b *RisqBuilding) findProductionItem(o *RisqOrder) *RisqBuildingProductionItem {
	for _, item := range b.production_queue {
		if item.order == o {
			return item
		}
	}
	return nil
}

func (b *RisqBuilding) orderReceivable(o *RisqOrder, risq *GameRisq) bool {
	return true
}

func (b *RisqBuilding) receiveOrder(o *RisqOrder, risq *GameRisq) {
	already_received := o.received
	b.order_queue.receiveOrder(o)
	if already_received {
		return
	}
	switch o.order_type {
	case OrderType_BuildingCreate:
		unit_id := uint32(o.target_id)
		cost, stamina_required := unitProductionCost(unit_id)
		risq.players[b.player_id].resources.spend(cost)
		b.production_queue = append(b.production_queue, &RisqBuildingProductionItem{
			order:             o,
			unit_id:           unit_id,
			stamina_remaining: stamina_required,
		})
	}
}

func (b *RisqBuilding) orderComplete(o *RisqOrder, risq *GameRisq) bool {
	switch o.order_type {
	case OrderType_BuildingCreate:
		item := b.findProductionItem(o)
		return item == nil || item.stamina_remaining <= 0
	default:
		return true
	}
}

func (b *RisqBuilding) tickIntent(risq *GameRisq) bool {
	b.intent.resetIntent()
	order := b.order_queue.nextOrder(b, risq)
	if order == nil {
		return false
	}
	switch order.order_type {
	case OrderType_BuildingCreate:
		b.intent.setProduce(b.findProductionItem(order))
	default:
		fmt.Fprintln(os.Stderr, "Order type not implemented:", order.order_type)
	}
	b.intent.resolveCost(b.current_stamina)
	if _, producing := b.intent.detail.(*ProductionIntent); producing && risq.players[b.player_id].populationCapped() {
		b.intent.resetIntent()
	}
	return b.intent.hasIntent()
}

func (b *RisqBuilding) tickExecute(risq *GameRisq) {
	if !b.intent.hasIntent() {
		return
	}
	if detail, ok := b.intent.detail.(*ProductionIntent); ok {
		if risq.players[b.player_id].populationCapped() {
			return
		}
		item := detail.item
		item.stamina_remaining -= b.intent.intent_cost
		if item.stamina_remaining <= 0 {
			unit := createRisqUnit(risq.nextUnitInternalId(), item.unit_id, b.player_id)
			b.zone.space.setUnit(&b.zone.coordinate, unit)
			risq.players[b.player_id].units[unit.internal_id] = unit
			b.production_queue = b.production_queue[1:]
		}
	}
	b.current_stamina -= b.intent.intent_cost
}

func (b *RisqBuilding) toFrontend() gin.H {
	building := gin.H{
		"internal_id":        b.internal_id,
		"player_id":          b.player_id,
		"building_id":        b.building_id,
		"display_name":       b.display_name,
		"population_support": b.population_support,
		"combat_stats":       b.cs.toFrontend(),
		"under_construction": b.underConstruction(),
		"stamina_remaining":  b.stamina_remaining,
	}
	if b.zone != nil {
		building["zone_coordinate"] = b.zone.coordinate.ToFrontend()
		if b.zone.space != nil {
			building["space_coordinate"] = b.zone.space.coordinate.ToFrontend()
		}
	}
	active_orders := make([]gin.H, 0)
	for _, order := range b.order_queue.active_orders {
		if order != nil && !order.executed {
			active_orders = append(active_orders, order.toFrontend())
		}
	}
	building["active_orders"] = active_orders
	production_queue := make([]gin.H, 0)
	for _, item := range b.production_queue {
		if item != nil {
			production_queue = append(production_queue, item.toFrontend())
		}
	}
	building["production_queue"] = production_queue
	return building
}
