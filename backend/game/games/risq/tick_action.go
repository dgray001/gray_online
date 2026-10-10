package risq

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

type tickVisibility map[int]uint8

type tickLocation struct {
	space, zone   game_utils.Coordinate2D
	garrisoned_in uint64
}

type tickTarget struct {
	kind        string
	internal_id uint64
	player_id   int
}

type tickOrder struct {
	source_actor tickTarget
	internal_id  uint64
	order_type   defs.OrderType
	target_id    int64
	source       string
}

type tickResolution struct {
	kind, reason                 string
	stamina_allocated, sunk_cost int
}

type tickIntent struct {
	item_id         uint32
	producible_kind defs.ProducibleKind

	kind                                  string
	location                              tickLocation
	destination, target_location          *tickLocation
	target                                tickTarget
	available_stamina, min_cost, max_cost int
	resolution                            tickResolution
}

type tickExecution struct {
	kind string

	outcome, reason         string
	stamina_spent, progress int
	target                  tickTarget
	target_location         *tickLocation
	gathered, healing       float64
	cost                    defs.RisqResourceCost
	effects                 []tickEffect
}

type TickAction struct {
	execute_target_visibility tickVisibility

	tick                          uint16
	sequence                      int
	order                         *tickOrder
	intent                        tickIntent
	execute                       tickExecution
	player_id                     int
	visibility, target_visibility tickVisibility
}

type tickEffect struct {
	reason  string
	order   *tickOrder
	private bool

	id          uint64
	tick        uint16
	damage_type defs.AttackType

	kind                                string
	actor, target                       tickTarget
	amount                              float64
	actor_visibility, target_visibility tickVisibility
}

func tickBase(actor Orderable) *orderableBase {
	switch actor := actor.(type) {
	case *RisqUnit:
		return &actor.orderableBase
	case *RisqBuilding:
		return &actor.orderableBase
	case garrisonAttacker:
		return &actor.RisqUnit.orderableBase
	default:
		panic("unknown tick actor")
	}
}

func tickActorZone(actor Orderable) *RisqZone {
	if unit, ok := actor.(*RisqUnit); ok && unit.garrisoned_in != nil {
		return unit.garrisoned_in.zone
	}
	return tickBase(actor).zone
}

func tickActorTarget(actor Orderable) tickTarget {
	kind := "unit"
	if actor.OrderableType() == defs.OrderableType_BUILDING {
		kind = "building"
	}
	return tickTarget{kind: kind, internal_id: actor.internalId(), player_id: tickBase(actor).player_id}
}

func tickZoneLocation(zone *RisqZone) tickLocation {
	if zone == nil {
		return tickLocation{}
	}
	return tickLocation{space: zone.space.coordinate, zone: zone.coordinate}
}

func tickActorLocation(actor Orderable) tickLocation {
	location := tickZoneLocation(tickActorZone(actor))
	if unit, ok := actor.(*RisqUnit); ok && unit.garrisoned_in != nil {
		location.garrisoned_in = unit.garrisoned_in.internal_id
	}
	return location
}

func tickZoneVisibility(zone *RisqZone) tickVisibility {
	visibility := tickVisibility{}
	if zone != nil {
		for player_id, level := range zone.space.visibility {
			visibility[player_id] = max(level, zone.space.getVisibility(player_id))
		}
	}
	return visibility
}

func tickOrderValue(order *RisqOrder) *tickOrder {
	if order == nil {
		return nil
	}
	source := order.tick_source
	if source == "" {
		source = "player"
	}
	return &tickOrder{internal_id: order.internal_id, order_type: order.order_type, target_id: order.target_id, source: source}
}
