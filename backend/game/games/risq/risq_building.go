package risq

import (
	"fmt"
	"os"

	"github.com/dgray001/gray_online/util"
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
	production_queue   map[uint64]*RisqBuildingProductionItem
	intent             *RisqIntent
	// build stamina still needed to finish a unit-constructed foundation; 0 means not under construction
	stamina_remaining          int
	construction_stamina_total int
}

func (b *RisqBuilding) underConstruction() bool {
	return b.stamina_remaining > 0
}

const constructionMinHealthRatio = 0.1

func constructionHealthRatio(stamina_remaining int, construction_stamina_total int) float64 {
	if construction_stamina_total <= 0 {
		return 1
	}
	progress := util.Clamp(1-float64(stamina_remaining)/float64(construction_stamina_total), 0.0, 1.0)
	return constructionMinHealthRatio + (1-constructionMinHealthRatio)*progress
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
		production_queue:   make(map[uint64]*RisqBuildingProductionItem),
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
	v := buildingConfigs[b.building_id].vision
	return &v
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
	item_id           uint32
	stamina_remaining int
	cost              RisqResourceCost
}

func (item *RisqBuildingProductionItem) toFrontend() gin.H {
	return gin.H{
		"item_id":           item.item_id,
		"stamina_remaining": item.stamina_remaining,
	}
}

func (b *RisqBuilding) orderReceivable(o *RisqOrder, risq *GameRisq) bool {
	return !b.underConstruction()
}

func (b *RisqBuilding) receiveOrder(o *RisqOrder, risq *GameRisq) {
	b.order_queue.receiveOrder(o)
	switch o.order_type {
	case OrderType_BuildingCreate:
		unit_id := uint32(o.target_id)
		cost, stamina_required := unitProductionCost(unit_id)
		resources := risq.players[b.player_id].resources
		if !resources.canAfford(cost) {
			// TODO: surface this failure in the per-player turn report
			return
		}
		resources.spend(cost)
		b.production_queue[o.internal_id] = &RisqBuildingProductionItem{
			item_id:           unit_id,
			stamina_remaining: stamina_required,
			cost:              cost,
		}
	}
}

func (b *RisqBuilding) cancelOrder(o *RisqOrder, risq *GameRisq) {
	b.order_queue.cancelOrder(o.internal_id)
	switch o.order_type {
	case OrderType_BuildingCreate:
		item, ok := b.production_queue[o.internal_id]
		if !ok {
			return
		}
		risq.players[b.player_id].resources.refund(item.cost)
		delete(b.production_queue, o.internal_id)
	}
}

func (b *RisqBuilding) orderStatus(o *RisqOrder, risq *GameRisq) OrderStatus {
	switch o.order_type {
	case OrderType_BuildingCreate:
		if item, ok := b.production_queue[o.internal_id]; ok && item.stamina_remaining > 0 {
			return OrderStatus_InProgress
		}
	}
	return OrderStatus_Executed
}

func (b *RisqBuilding) tickIntent(risq *GameRisq) bool {
	b.intent.resetIntent()
	order := b.order_queue.nextOrder(b, risq)
	if order == nil {
		return false
	}
	switch order.order_type {
	case OrderType_BuildingCreate:
		b.intent.setCreate(order.internal_id, b.production_queue[order.internal_id])
	default:
		fmt.Fprintln(os.Stderr, "Order type not implemented:", order.order_type)
	}
	b.intent.resolveCost(b.current_stamina)
	if _, creating := b.intent.detail.(*CreateIntent); creating && risq.players[b.player_id].populationCapped() {
		b.intent.resetIntent()
	}
	return b.intent.hasIntent()
}

func (b *RisqBuilding) tickExecute(risq *GameRisq) {
	if !b.intent.hasIntent() {
		return
	}
	if detail, ok := b.intent.detail.(*CreateIntent); ok {
		if risq.players[b.player_id].populationCapped() {
			return
		}
		item := detail.item
		item.stamina_remaining -= b.intent.intent_cost
		if item.stamina_remaining <= 0 {
			unit := createRisqUnit(risq.nextUnitInternalId(), item.item_id, b.player_id)
			b.zone.space.setUnit(&b.zone.coordinate, unit)
			risq.players[b.player_id].units[unit.internal_id] = unit
			delete(b.production_queue, detail.order_internal_id)
		}
	}
	b.current_stamina -= b.intent.intent_cost
}

func buildingProducesToFrontend(building_id uint32) []gin.H {
	produces := make([]gin.H, 0)
	for _, p := range buildingConfigs[building_id].produces {
		produces = append(produces, p.toFrontend())
	}
	return produces
}

func (b *RisqBuilding) toFrontend(viewer_player_id int) gin.H {
	building := gin.H{
		"internal_id":                b.internal_id,
		"player_id":                  b.player_id,
		"building_id":                b.building_id,
		"display_name":               b.display_name,
		"population_support":         b.population_support,
		"combat_stats":               b.cs.toFrontend(),
		"under_construction":         b.underConstruction(),
		"stamina_remaining":          b.stamina_remaining,
		"construction_stamina_total": b.construction_stamina_total,
		"turn_stamina":               b.turn_stamina,
		"current_stamina":            b.current_stamina,
		"max_stamina":                b.max_stamina,
	}
	building["produces"] = buildingProducesToFrontend(b.building_id)
	if b.zone != nil {
		building["zone_coordinate"] = b.zone.coordinate.ToFrontend()
		if b.zone.space != nil {
			building["space_coordinate"] = b.zone.space.coordinate.ToFrontend()
		}
	}
	active_orders := make([]gin.H, 0)
	production_queue := make([]gin.H, 0)
	if showOrdersTo(b.player_id, b.zone, viewer_player_id) {
		for _, order := range b.order_queue.active_orders {
			if order == nil || order.executed {
				continue
			}
			active_orders = append(active_orders, order.toFrontend())
			if item, ok := b.production_queue[order.internal_id]; ok {
				item_frontend := item.toFrontend()
				item_frontend["order_internal_id"] = order.internal_id
				production_queue = append(production_queue, item_frontend)
			}
		}
	}
	building["active_orders"] = active_orders
	building["production_queue"] = production_queue
	return building
}
