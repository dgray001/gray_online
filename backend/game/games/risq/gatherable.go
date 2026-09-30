package risq

import (
	"sort"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
)

type Gatherable interface {
	gatherCategory() defs.RisqResourceCategory
	gatherSpeed() int
	gatherResourcesLeft() float64
	gatherDrain(amount float64)
	gatherCapacity() int
}

func (r *RisqResource) gatherCapacity() int {
	return r.gather_capacity
}

func (b *RisqBuilding) gatherCapacity() int {
	return defs.BuildingConfigs[b.building_id].Gather.Gather_capacity
}

// The live source in a zone: its resource, or its gatherable building
func zoneGatherSource(zone *RisqZone) Gatherable {
	if zone.resource != nil {
		return zone.resource
	}
	if b := zone.building; b != nil && defs.BuildingConfigs[b.building_id].IsGatherable() {
		return b
	}
	return nil
}

func (u *RisqUnit) hasGatherOrderAt(zone *RisqZone, risq *GameRisq) bool {
	for _, o := range u.order_queue.active_orders {
		if o.order_type != defs.OrderType_UnitGather {
			continue
		}
		if _, z := invertZoneKey(uint(o.target_id), risq); z == zone {
			return true
		}
	}
	return false
}

// Units holding a gather slot at source (they gathered it last tick and still mean to), other than exclude.
// With viewer >= 0 only the holders that player can see count: its own, plus others where it has good vision.
func gatherSlotHolders(source Gatherable, zone *RisqZone, risq *GameRisq, viewer int, exclude *RisqUnit) int {
	count := 0
	for _, p := range risq.players {
		if viewer >= 0 && p.player.Player_id != viewer && zone.space.getVisibility(viewer) < defs.VisibilityGood {
			continue
		}
		for _, u := range p.units {
			if u != exclude && !u.deleted && u.gather_slot == source && u.hasGatherOrderAt(zone, risq) {
				count++
			}
		}
	}
	return count
}

func (r *RisqResource) gatherCategory() defs.RisqResourceCategory {
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

func (b *RisqBuilding) gatherCategory() defs.RisqResourceCategory {
	return defs.BuildingConfigs[b.building_id].Gather.Resource_category
}

func (b *RisqBuilding) gatherSpeed() int {
	return defs.BuildingConfigs[b.building_id].Gather.Base_gather_speed
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

// Applied after every gather drain in the tick, so a same-tick gather never sees this tick's refill
func (b *RisqBuilding) resolveRenew() {
	if b.pending_renew == 0 {
		return
	}
	starting_resources := defs.BuildingConfigs[b.building_id].Gather.Starting_resources
	b.resources_left = util.RoundTo(min(b.resources_left+b.pending_renew, starting_resources), 4)
	b.pending_renew = 0
	if b.resources_left >= starting_resources {
		b.renewing = nil
	}
	b.refreshTerrainOverride()
}

func (b *RisqBuilding) refreshTerrainOverride() {
	if b.zone == nil {
		return
	}
	config := defs.BuildingConfigs[b.building_id]
	terrain_type := b.zone.space.terrainType()
	override := config.Terrain_override[terrain_type]
	if config.IsGatherable() && b.resources_left <= 0 {
		if dead, ok := config.Gather.Terrain_override_dead[terrain_type]; ok {
			override = dead
		}
	}
	b.zone.terrain_override = override
}

func (r *GameRisq) autoGatherCompletedBuildings() {
	for _, b := range r.completed_gatherables {
		builders := make([]*RisqUnit, 0)
		for _, u := range r.players[b.player_id].units {
			construction, ok := u.intent.detail.(*ConstructionIntent)
			if ok && !u.deleted && construction.zone == b.zone && len(u.order_queue.active_orders) == 1 {
				builders = append(builders, u)
			}
		}
		sort.Slice(builders, func(i, j int) bool { return builders[i].internal_id < builders[j].internal_id })
		for _, u := range builders {
			order := createRisqOrder(r.nextOrderInternalId(), defs.OrderType_UnitGather, b.player_id, map[uint64]Orderable{u.internal_id: u}, int64(b.zone.coordinate_key), false)
			if u.orderReceivable(order, r) {
				u.receiveOrder(order, r)
			}
		}
	}
	r.completed_gatherables = r.completed_gatherables[:0]
}
