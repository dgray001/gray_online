package risq

import (
	"iter"

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
	active_orders        []*RisqOrder
	past_orders          []*RisqOrder
	orders_submitted     bool
	planned_foundations  map[uint]*RisqPlannedFoundation
}

// Private commitment to build at a zone before any stamina makes it a real, objective RisqBuilding
type RisqPlannedFoundation struct {
	building_id    uint32
	cost           RisqResourceCost
	creating_order *RisqOrder
}

func createRisqPlayer(player *game.Player, max_population_limit uint16, color string) *RisqPlayer {
	return &RisqPlayer{
		player:               player,
		resources:            createRisqPlayerResources(),
		buildings:            make(map[uint64]*RisqBuilding),
		units:                make(map[uint64]*RisqUnit, 0),
		max_population_limit: max_population_limit,
		color:                color,
		active_orders:        make([]*RisqOrder, 0),
		past_orders:          make([]*RisqOrder, 0),
		orders_submitted:     false,
		planned_foundations:  make(map[uint]*RisqPlannedFoundation),
	}
}

func createRisqPlannedFoundation(building_id uint32, creating_order *RisqOrder, player *RisqPlayer) *RisqPlannedFoundation {
	cost, _ := buildingProductionCost(building_id)
	player.resources.spend(cost)
	return &RisqPlannedFoundation{building_id: building_id, cost: cost, creating_order: creating_order}
}

func (p *RisqPlayer) cancelPlannedFoundation(zone *RisqZone) {
	foundation, ok := p.planned_foundations[zone.coordinate_key]
	if !ok {
		return
	}
	p.resources.refund(foundation.cost)
	delete(p.planned_foundations, zone.coordinate_key)
}

func (p *RisqPlayer) populationLimit() uint16 {
	limit := uint16(0)
	for _, building := range p.buildings {
		if building != nil && !building.deleted && !building.underConstruction() {
			limit += building.population_support
		}
	}
	if limit > p.max_population_limit {
		limit = p.max_population_limit
	}
	return limit
}

func (p *RisqPlayer) populationCapped() bool {
	return uint16(len(p.units)) >= p.populationLimit()
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

func (p *RisqPlayer) allOrderables() iter.Seq[Orderable] {
	return func(yield func(Orderable) bool) {
		for _, u := range p.units {
			if u.isDeleted() {
				continue
			}
			if !yield(u) {
				return
			}
		}
		for _, b := range p.buildings {
			if b.isDeleted() {
				continue
			}
			if !yield(b) {
				return
			}
		}
	}
}

func (p *RisqPlayer) receivePlayerOrder(o *RisqOrder, risq *GameRisq) {
	switch o.order_type {
	case OrderType_CancelOrder:
		for _, active_order := range p.active_orders {
			if active_order.internal_id != uint64(o.target_id) {
				continue
			}
			for _, subject := range active_order.subjects {
				subject.cancelOrder(active_order, risq)
			}
			break
		}
	case OrderType_CancelFoundation:
		_, zone := invertZoneKey(uint(o.target_id), risq)
		p.cancelPlannedFoundation(zone)
	}
}

func (p *RisqPlayer) toFrontend(viewer_player_id int) gin.H {
	player := gin.H{
		"population_limit": p.populationLimit(),
		"score":            p.score(),
		"color":            p.color,
		"orders_submitted": p.orders_submitted,
	}
	if p.player != nil {
		player["player"] = p.player.ToFrontend(false)
	}
	if p.resources != nil && p.player != nil && p.player.Player_id == viewer_player_id {
		player["resources"] = p.resources.toFrontend()
	}
	buildings := make([]gin.H, 0)
	for _, building := range p.buildings {
		if building != nil && !building.deleted {
			buildings = append(buildings, building.toFrontend(viewer_player_id))
		}
	}
	player["buildings"] = buildings
	units := make([]gin.H, 0)
	for _, unit := range p.units {
		if unit != nil && !unit.deleted {
			units = append(units, unit.toFrontend(viewer_player_id))
		}
	}
	player["units"] = units
	active_orders := make([]gin.H, 0)
	if p.player != nil && p.player.Player_id == viewer_player_id {
		for _, order := range p.active_orders {
			if order != nil && !order.executed && !order.cancelled {
				active_orders = append(active_orders, order.toFrontend())
			}
		}
	}
	player["active_orders"] = active_orders
	return player
}
