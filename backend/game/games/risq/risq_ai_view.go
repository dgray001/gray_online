package risq

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/util"
)

type aiView struct {
	player            *RisqPlayer
	risq              *GameRisq
	claimed_units     map[uint64]bool
	claimed_buildings map[uint64]bool
}

func newAiView(p *RisqPlayer, r *GameRisq) ai.View {
	return &aiView{player: p, risq: r, claimed_units: map[uint64]bool{}, claimed_buildings: map[uint64]bool{}}
}

func (v *aiView) Nickname() string {
	return v.player.player.GetNickname()
}

func toCoordinate(c game_utils.Coordinate2D) ai.Coordinate {
	return ai.Coordinate{X: c.X, Y: c.Y}
}

func toZoneRef(z *RisqZone) ai.ZoneRef {
	return ai.ZoneRef{Space: toCoordinate(z.space.coordinate), Zone: toCoordinate(z.coordinate)}
}

func toCost(c RisqResourceCost) ai.Cost {
	return ai.Cost{Food: c.food, Wood: c.wood, Stone: c.stone, Gold: c.gold}
}

func (v *aiView) resolveZone(ref ai.ZoneRef) (*RisqSpace, *RisqZone) {
	space := v.risq.getSpace(&game_utils.Coordinate2D{X: ref.Space.X, Y: ref.Space.Y})
	if space == nil {
		return nil, nil
	}
	return space, space.getZone(&game_utils.Coordinate2D{X: ref.Zone.X, Y: ref.Zone.Y})
}

func isEconomicUnit(unit_id uint32) bool {
	return unit_id < 11
}

func toUnitView(u *RisqUnit) ai.UnitView {
	kind := ai.UnitMilitary
	if isEconomicUnit(u.unit_id) {
		kind = ai.UnitEconomic
	}
	return ai.UnitView{
		InternalID:     u.internal_id,
		UnitID:         u.unit_id,
		Kind:           kind,
		Location:       toZoneRef(u.zone),
		CurrentStamina: u.current_stamina,
		Idle:           len(u.order_queue.active_orders) == 0,
	}
}

func buildingIdle(b *RisqBuilding) bool {
	return !b.underConstruction() && len(b.order_queue.active_orders) == 0 && len(buildingConfigs[b.building_id].produces) > 0
}

func buildingProducibles(b *RisqBuilding, risq *GameRisq) []ai.Producible {
	owner := risq.players[b.player_id]
	producibles := make([]ai.Producible, 0)
	for _, p := range buildingConfigs[b.building_id].produces {
		switch p.kind {
		case ProducibleKind_UNIT:
			cost, _ := unitProductionCost(p.id)
			producibles = append(producibles, ai.Producible{Kind: ai.ProducibleUnit, ID: p.id, Cost: toCost(cost)})
		case ProducibleKind_TECH:
			if _, researching_or_done := owner.researched_techs[p.id]; researching_or_done {
				continue
			}
			producibles = append(producibles, ai.Producible{Kind: ai.ProducibleTech, ID: p.id, Cost: toCost(techConfigs[p.id].cost)})
		}
	}
	return producibles
}

func toBuildingView(b *RisqBuilding, risq *GameRisq) ai.BuildingView {
	return ai.BuildingView{
		InternalID:        b.internal_id,
		BuildingID:        b.building_id,
		Location:          toZoneRef(b.zone),
		UnderConstruction: b.underConstruction(),
		Idle:              buildingIdle(b),
		Producibles:       buildingProducibles(b, risq),
	}
}

func (v *aiView) claimUnit(id uint64) {
	v.claimed_units[id] = true
}

func (v *aiView) claimBuilding(id uint64) {
	v.claimed_buildings[id] = true
}

func (v *aiView) Units() []ai.UnitView {
	units := make([]ai.UnitView, 0, len(v.player.units))
	for _, u := range v.player.units {
		if u != nil && !u.deleted {
			units = append(units, toUnitView(u))
		}
	}
	return units
}

func (v *aiView) IdleUnits() []ai.UnitView {
	units := make([]ai.UnitView, 0)
	for _, u := range v.player.units {
		if u != nil && !u.deleted && len(u.order_queue.active_orders) == 0 && !v.claimed_units[u.internal_id] {
			units = append(units, toUnitView(u))
		}
	}
	return units
}

func (v *aiView) Buildings() []ai.BuildingView {
	buildings := make([]ai.BuildingView, 0, len(v.player.buildings))
	for _, b := range v.player.buildings {
		if b != nil && !b.deleted {
			buildings = append(buildings, toBuildingView(b, v.risq))
		}
	}
	return buildings
}

func (v *aiView) IdleBuildings() []ai.BuildingView {
	buildings := make([]ai.BuildingView, 0)
	for _, b := range v.player.buildings {
		if b != nil && !b.deleted && buildingIdle(b) && !v.claimed_buildings[b.internal_id] {
			buildings = append(buildings, toBuildingView(b, v.risq))
		}
	}
	return buildings
}

func (v *aiView) Resource(category ai.ResourceCategory) float64 {
	switch RisqResourceCategory(category) {
	case RisqResourceCategory_FOOD:
		return v.player.resources.food
	case RisqResourceCategory_WOOD:
		return v.player.resources.wood
	case RisqResourceCategory_STONE:
		return v.player.resources.stone
	case RisqResourceCategory_GOLD:
		return v.player.resources.gold
	default:
		return 0
	}
}

func (v *aiView) Population() (int, int) {
	return nonDeletedUnitCount(v.player.units), int(v.player.populationLimit())
}

func (v *aiView) playerId() int {
	return v.player.player.Player_id
}

func (v *aiView) VisibleEnemyUnits() []ai.UnitView {
	units := make([]ai.UnitView, 0)
	for _, player := range v.risq.players {
		if player == v.player {
			continue
		}
		for _, u := range player.units {
			if u == nil || u.deleted || u.zone == nil || u.zone.space == nil {
				continue
			}
			if u.zone.space.getVisibility(v.playerId()) >= VisibilityGood {
				units = append(units, toUnitView(u))
			}
		}
	}
	return units
}

func (v *aiView) VisibleEnemyBuildings() []ai.BuildingView {
	buildings := make([]ai.BuildingView, 0)
	for _, player := range v.risq.players {
		if player == v.player {
			continue
		}
		for _, b := range player.buildings {
			if b == nil || b.deleted || b.zone == nil || b.zone.space == nil {
				continue
			}
			if b.zone.space.getVisibility(v.playerId()) >= VisibilityPoor {
				buildings = append(buildings, toBuildingView(b, v.risq))
			}
		}
	}
	return buildings
}

func (v *aiView) NearestResource(from ai.ZoneRef, category ai.ResourceCategory) (ai.ZoneRef, bool) {
	from_space, from_zone := v.resolveZone(from)
	if from_space == nil {
		return ai.ZoneRef{}, false
	}
	var best *RisqZone
	var best_distance uint
	for _, row := range v.risq.spaces {
		for _, space := range row {
			if space.getVisibility(v.playerId()) < VisibilityFog {
				continue
			}
			distance := game_utils.AxialDistance(from_space.coordinate, space.coordinate) * 6
			for _, zone_row := range space.zones {
				for _, zone := range zone_row {
					if zone.resource == nil || zone.resource.resources_left <= 0 || zone.resource.category() != RisqResourceCategory(category) {
						continue
					}
					d := distance
					if space == from_space {
						d = uint(zoneDistanceWithinSpace(from_zone, zone))
					}
					if best == nil || d < best_distance {
						best, best_distance = zone, d
					}
				}
			}
		}
	}
	if best == nil {
		return ai.ZoneRef{}, false
	}
	return toZoneRef(best), true
}

func (v *aiView) BuildCost(building_id uint32) ai.Cost {
	cost, _ := buildingProductionCost(building_id)
	return toCost(cost)
}

func (v *aiView) NearestUnexplored(from ai.ZoneRef) (ai.ZoneRef, bool) {
	from_space, _ := v.resolveZone(from)
	if from_space == nil {
		return ai.ZoneRef{}, false
	}
	var best *RisqSpace
	var best_distance uint
	for _, row := range v.risq.spaces {
		for _, space := range row {
			if space.getVisibility(v.playerId()) != VisibilityUnexplored {
				continue
			}
			d := game_utils.AxialDistance(from_space.coordinate, space.coordinate)
			if best == nil || d < best_distance {
				best, best_distance = space, d
			}
		}
	}
	if best == nil {
		return ai.ZoneRef{}, false
	}
	return toZoneRef(best.getCenterZone()), true
}

func (v *aiView) NearestBuildSite(from ai.ZoneRef, building_id uint32) (ai.ZoneRef, bool) {
	from_space, from_zone := v.resolveZone(from)
	if from_space == nil {
		return ai.ZoneRef{}, false
	}
	var best *RisqZone
	var best_distance uint
	for _, row := range v.risq.spaces {
		for _, space := range row {
			if space.ownership >= 0 && space.ownership != v.playerId() {
				continue
			}
			distance := game_utils.AxialDistance(from_space.coordinate, space.coordinate) * 6
			for _, zone_row := range space.zones {
				for _, zone := range zone_row {
					if zone.resource != nil || zone.building != nil {
						continue
					}
					d := distance
					if space == from_space {
						d = uint(zoneDistanceWithinSpace(from_zone, zone))
					}
					if best == nil || d < best_distance {
						best, best_distance = zone, d
					}
				}
			}
		}
	}
	if best == nil {
		return ai.ZoneRef{}, false
	}
	return toZoneRef(best), true
}

func (v *aiView) MoveOrder(u ai.UnitView, target ai.ZoneRef, clear_previous bool) ai.Order {
	v.claimUnit(u.InternalID)
	_, zone := v.resolveZone(target)
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: uint8(OrderType_UnitMoveZone), TargetID: int64(zone.coordinate_key), ClearPreviousOrders: clear_previous}
}

func (v *aiView) GatherOrder(u ai.UnitView, target ai.ZoneRef, clear_previous bool) ai.Order {
	v.claimUnit(u.InternalID)
	_, zone := v.resolveZone(target)
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: uint8(OrderType_UnitGather), TargetID: int64(zone.coordinate_key), ClearPreviousOrders: clear_previous}
}

func (v *aiView) BuildOrder(u ai.UnitView, building_id uint32, target ai.ZoneRef, clear_previous bool) ai.Order {
	v.claimUnit(u.InternalID)
	_, zone := v.resolveZone(target)
	target_id := util.Pair(int(building_id), int(zone.coordinate_key))
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: uint8(OrderType_UnitBuild), TargetID: int64(target_id), ClearPreviousOrders: clear_previous}
}

func (v *aiView) RepairOrder(u ai.UnitView, target_building_id uint64, clear_previous bool) ai.Order {
	v.claimUnit(u.InternalID)
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: uint8(OrderType_UnitRepair), TargetID: int64(target_building_id), ClearPreviousOrders: clear_previous}
}

func (v *aiView) AttackUnitOrder(u ai.UnitView, target_unit_id uint64, clear_previous bool) ai.Order {
	v.claimUnit(u.InternalID)
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: uint8(OrderType_UnitAttackUnit), TargetID: int64(target_unit_id), ClearPreviousOrders: clear_previous}
}

func (v *aiView) AttackBuildingOrder(u ai.UnitView, target_building_id uint64, clear_previous bool) ai.Order {
	v.claimUnit(u.InternalID)
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: uint8(OrderType_UnitAttackBuilding), TargetID: int64(target_building_id), ClearPreviousOrders: clear_previous}
}

func (v *aiView) DeleteUnitOrder(u ai.UnitView) ai.Order {
	v.claimUnit(u.InternalID)
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: uint8(OrderType_UnitDelete), ClearPreviousOrders: true}
}

func (v *aiView) CreateUnitOrder(b ai.BuildingView, unit_id uint32) ai.Order {
	v.claimBuilding(b.InternalID)
	return ai.Order{Subjects: []uint64{b.InternalID}, OrderType: uint8(OrderType_BuildingCreate), TargetID: int64(unit_id)}
}

func (v *aiView) ResearchOrder(b ai.BuildingView, tech_id uint32) ai.Order {
	v.claimBuilding(b.InternalID)
	return ai.Order{Subjects: []uint64{b.InternalID}, OrderType: uint8(OrderType_BuildingResearch), TargetID: int64(tech_id)}
}
