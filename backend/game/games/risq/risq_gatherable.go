package risq

import "github.com/dgray001/gray_online/util"

type Gatherable interface {
	gatherCategory() RisqResourceCategory
	gatherSpeed() int
	gatherResourcesLeft() float64
	gatherDrain(amount float64)
}

func (r *RisqResource) gatherCategory() RisqResourceCategory {
	return r.category()
}

func (r *RisqResource) gatherSpeed() int {
	return r.base_gather_speed
}

func (r *RisqResource) gatherResourcesLeft() float64 {
	return r.resources_left
}

func (r *RisqResource) gatherDrain(amount float64) {
	r.resources_left -= amount
	r.resources_left = util.RoundTo(r.resources_left, 4)
	if r.resources_left <= 0 && r.zone != nil {
		if r.zone.space != nil {
			delete(r.zone.space.resources, r.internal_id)
		}
		r.zone.resource = nil
	}
}

func (b *RisqBuilding) gatherCategory() RisqResourceCategory {
	return buildingConfigs[b.building_id].gather.resource_category
}

func (b *RisqBuilding) gatherSpeed() int {
	return buildingConfigs[b.building_id].gather.base_gather_speed
}

func (b *RisqBuilding) gatherResourcesLeft() float64 {
	return b.resources_left
}

func (b *RisqBuilding) gatherDrain(amount float64) {
	b.resources_left -= amount
	b.resources_left = util.RoundTo(b.resources_left, 4)
	if b.resources_left < 0 {
		b.resources_left = 0
	}
	b.refreshTerrainOverride()
}

func (b *RisqBuilding) refreshTerrainOverride() {
	if b.zone == nil {
		return
	}
	config := buildingConfigs[b.building_id]
	terrain_type := b.zone.space.terrainType()
	override := config.terrain_override[terrain_type]
	if config.isGatherable() && b.resources_left <= 0 {
		if dead, ok := config.gather.terrain_override_dead[terrain_type]; ok {
			override = dead
		}
	}
	b.zone.terrain_override = override
}

// Counts units currently gathering from this building, derived live from active orders
func (b *RisqBuilding) gatheringUnitCount(risq *GameRisq) int {
	count := 0
	for _, u := range risq.players[b.player_id].units {
		for _, o := range u.order_queue.active_orders {
			if o.order_type != OrderType_UnitGather {
				continue
			}
			if _, zone := invertZoneKey(uint(o.target_id), risq); zone == b.zone {
				count++
				break
			}
		}
	}
	return count
}
