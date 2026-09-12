package risq

import (
	"embed"
	"encoding/json"
	"iter"
	"math/rand"
	"path"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/gin-gonic/gin"
)

//go:embed config/ai/*
var aiConfigs embed.FS

//go:embed config/ai/default.json
var defaultAiConfig []byte

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
	researched_techs     map[uint32]bool
	report               *RisqTurnReport
	score                uint
	ai_model             ai.Model
	eliminated           bool
	kills                uint
	razes                uint
	units_lost           uint
	buildings_lost       uint
	// owned by this player only, so its own AI goroutine never races another player's
	rng *rand.Rand
}

func (p *RisqPlayer) createAiModel(config_path string) {
	targetPath := path.Join("config/ai", config_path+".json")
	data, read_err := aiConfigs.ReadFile(targetPath)
	if read_err != nil {
		// TODO: log error
		data = defaultAiConfig
	}
	var raw map[string]any
	unmarshal_err := json.Unmarshal(data, &raw)
	if unmarshal_err != nil {
		// TODO: log error
		p.ai_model = ai.ParseModel(nil)
		return
	}
	p.ai_model = ai.ParseModel(raw)
}

// Private commitment to build at a zone before any stamina makes it a real, objective RisqBuilding
type RisqPlannedFoundation struct {
	building_id uint32
	cost        RisqResourceCost
}

func createRisqPlayer(player *game.Player, max_population_limit uint16, color string, rng *rand.Rand) *RisqPlayer {
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
		researched_techs:     make(map[uint32]bool),
		rng:                  rng,
	}
}

func createRisqPlannedFoundation(building_id uint32, player *RisqPlayer) *RisqPlannedFoundation {
	cost, _ := buildingProductionCost(building_id)
	player.resources.spend(cost)
	return &RisqPlannedFoundation{building_id: building_id, cost: cost}
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
	return uint16(nonDeletedUnitCount(p.units)) >= p.populationLimit()
}

func (p *RisqPlayer) valid() bool {
	return true
}

func (p *RisqPlayer) canSubmitOrders() bool {
	return !p.eliminated
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
		"score":            p.score,
		"color":            p.color,
		"orders_submitted": p.orders_submitted,
		"eliminated":       p.eliminated,
	}
	if p.player != nil {
		player["player"] = p.player.ToFrontend(false)
	}
	if p.resources != nil && p.player != nil && p.player.Player_id == viewer_player_id {
		player["resources"] = p.resources.toFrontend()
		player["turn_report"] = p.report.toFrontend()
		foundations := make([]gin.H, 0)
		for coordinate_key, f := range p.planned_foundations {
			foundations = append(foundations, gin.H{
				"coordinate_key": coordinate_key,
				"building_id":    f.building_id,
				"display_name":   buildingConfigs[f.building_id].display_name,
			})
		}
		player["planned_foundations"] = foundations
	}
	is_owner := p.player != nil && p.player.Player_id == viewer_player_id
	buildings := make([]gin.H, 0)
	for _, building := range p.buildings {
		if building == nil || building.deleted {
			continue
		}
		if !is_owner && (building.zone == nil || building.zone.space == nil || building.zone.space.getVisibility(viewer_player_id) < VisibilityPoor) {
			continue
		}
		buildings = append(buildings, building.toFrontend(viewer_player_id))
	}
	player["buildings"] = buildings
	units := make([]gin.H, 0)
	for _, unit := range p.units {
		if unit == nil || unit.deleted {
			continue
		}
		if !is_owner && (unit.zone == nil || unit.zone.space == nil || unit.zone.space.getVisibility(viewer_player_id) < VisibilityGood) {
			continue
		}
		units = append(units, unit.toFrontend(viewer_player_id))
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
	player["researched_techs"] = p.researched_techs
	return player
}
