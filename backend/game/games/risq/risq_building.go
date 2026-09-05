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
	building.turn_stamina = config.turn_stamina
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

func (b *RisqBuilding) delete(risq *GameRisq) {
	delete(risq.players[b.player_id].buildings, b.internal_id)
	delete(risq.buildings, b.internal_id)
	if b.zone != nil && b.zone.space != nil {
		b.zone.space.removeBuilding(b)
	}
	b.deleted = true
}

func (b *RisqBuilding) refreshStamina() {
	b.current_stamina += b.turn_stamina
	max_stamina := maxStaminaFor(b.turn_stamina)
	if b.current_stamina > max_stamina {
		b.current_stamina = max_stamina
	}
}

type RisqBuildingProductionItem struct {
	kind              ProducibleKind
	item_id           uint32
	stamina_remaining int
	cost              RisqResourceCost
}

func (item *RisqBuildingProductionItem) toFrontend() gin.H {
	return gin.H{
		"kind":              item.kind,
		"item_id":           item.item_id,
		"stamina_remaining": item.stamina_remaining,
	}
}

func (b *RisqBuilding) orderReceivable(o *RisqOrder, risq *GameRisq) bool {
	if o.order_type == OrderType_BuildingDelete {
		return true
	}
	if o.order_type == OrderType_BuildingResearch {
		tech_id := uint32(o.target_id)
		if _, exists := risq.players[b.player_id].researched_techs[tech_id]; exists {
			return false
		}
	}
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
			kind:              ProducibleKind_UNIT,
			item_id:           unit_id,
			stamina_remaining: stamina_required,
			cost:              cost,
		}
	case OrderType_BuildingResearch:
		tech_id := uint32(o.target_id)
		tech := techConfigs[tech_id]
		resources := risq.players[b.player_id].resources
		if !resources.canAfford(tech.cost) {
			// TODO: surface this failure in the per-player turn report
			return
		}
		resources.spend(tech.cost)
		b.production_queue[o.internal_id] = &RisqBuildingProductionItem{
			kind:              ProducibleKind_TECH,
			item_id:           tech_id,
			stamina_remaining: tech.research_stamina,
			cost:              tech.cost,
		}
		risq.players[b.player_id].researched_techs[tech_id] = false
	}
}

func (b *RisqBuilding) cancelOrder(o *RisqOrder, risq *GameRisq) {
	b.order_queue.cancelOrder(o.internal_id)
	item, ok := b.production_queue[o.internal_id]
	if !ok {
		return
	}
	risq.players[b.player_id].resources.refund(item.cost)
	delete(b.production_queue, o.internal_id)
	if item.kind == ProducibleKind_TECH {
		delete(risq.players[b.player_id].researched_techs, item.item_id)
	}
}

func (b *RisqBuilding) orderStatus(o *RisqOrder, risq *GameRisq) OrderStatus {
	switch o.order_type {
	case OrderType_BuildingCreate:
		if item, ok := b.production_queue[o.internal_id]; ok && item.stamina_remaining > 0 {
			return OrderStatus_InProgress
		}
	case OrderType_BuildingResearch:
		if item, ok := b.production_queue[o.internal_id]; ok && item.stamina_remaining > 0 {
			return OrderStatus_InProgress
		}
	case OrderType_BuildingDelete:
		if !b.deleted {
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
		b.intent.setProduction(order.internal_id, b.production_queue[order.internal_id])
	case OrderType_BuildingResearch:
		b.intent.setProduction(order.internal_id, b.production_queue[order.internal_id])
	case OrderType_BuildingDelete:
		b.intent.setDelete()
	default:
		fmt.Fprintln(os.Stderr, "Order type not implemented:", order.order_type)
	}
	b.intent.resolveCost(b.current_stamina)
	if production, ok := b.intent.detail.(*ProductionIntent); ok && production.item.kind == ProducibleKind_UNIT && risq.players[b.player_id].populationCapped() {
		b.intent.resetIntent()
	}
	return b.intent.hasIntent()
}

func (b *RisqBuilding) tickExecute(risq *GameRisq) {
	if !b.intent.hasIntent() {
		return
	}
	if detail, ok := b.intent.detail.(*ProductionIntent); ok {
		item := detail.item
		if item.kind == ProducibleKind_UNIT && risq.players[b.player_id].populationCapped() {
			return
		}
		item.stamina_remaining -= b.intent.intent_cost
		if item.stamina_remaining <= 0 {
			switch item.kind {
			case ProducibleKind_UNIT:
				unit := createRisqUnit(risq.nextUnitInternalId(), item.item_id, risq.players[b.player_id])
				b.zone.space.setUnit(&b.zone.coordinate, unit)
				risq.players[b.player_id].units[unit.internal_id] = unit
				risq.units[unit.internal_id] = unit
			case ProducibleKind_TECH:
				risq.completeResearch(risq.players[b.player_id], item.item_id)
			}
			delete(b.production_queue, detail.order_internal_id)
		}
	}
	if _, ok := b.intent.detail.(*DeleteIntent); ok {
		b.delete(risq)
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
		"max_stamina":                maxStaminaFor(b.turn_stamina),
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
