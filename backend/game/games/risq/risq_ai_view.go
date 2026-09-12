package risq

import (
	"sort"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/util"
)

type aiView struct {
	player            *RisqPlayer
	risq              *GameRisq
	assigned_units    map[uint64]*ai.CurrentOrder
	claimed_buildings map[uint64]bool
}

func newAiView(p *RisqPlayer, r *GameRisq) ai.View {
	return &aiView{player: p, risq: r, assigned_units: map[uint64]*ai.CurrentOrder{}, claimed_buildings: map[uint64]bool{}}
}

func (v *aiView) Nickname() string {
	return v.player.player.GetNickname()
}

func (v *aiView) RandomIntn(n int) int {
	return v.player.rng.Intn(n)
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

func toAiCategory(c RisqResourceCategory) ai.ResourceCategory {
	switch c {
	case RisqResourceCategory_FOOD:
		return ai.ResourceFood
	case RisqResourceCategory_WOOD:
		return ai.ResourceWood
	case RisqResourceCategory_STONE:
		return ai.ResourceStone
	default:
		return ai.ResourceGold
	}
}

func toOrderKind(order_type OrderType) (ai.OrderKind, bool) {
	switch order_type {
	case OrderType_UnitMoveSpace, OrderType_UnitMoveZone:
		return ai.OrderKindMove, true
	case OrderType_UnitGather:
		return ai.OrderKindGather, true
	case OrderType_UnitBuild:
		return ai.OrderKindBuild, true
	case OrderType_UnitRepair:
		return ai.OrderKindRepair, true
	case OrderType_UnitAttackSpace:
		return ai.OrderKindAttackSpace, true
	case OrderType_UnitAttackZone:
		return ai.OrderKindAttackZone, true
	case OrderType_UnitAttackUnit:
		return ai.OrderKindAttackUnit, true
	case OrderType_UnitAttackBuilding:
		return ai.OrderKindAttackBuilding, true
	case OrderType_UnitDefend:
		return ai.OrderKindDefend, true
	case OrderType_UnitGarrison:
		return ai.OrderKindGarrison, true
	case OrderType_UnitUngarrison:
		return ai.OrderKindUngarrison, true
	case OrderType_UnitDelete:
		return ai.OrderKindDelete, true
	default:
		return 0, false
	}
}

func currentOrder(u *RisqUnit, risq *GameRisq) *ai.CurrentOrder {
	if len(u.order_queue.active_orders) == 0 {
		return nil
	}
	active := u.order_queue.active_orders[0]
	kind, ok := toOrderKind(active.order_type)
	if !ok {
		return nil
	}
	order := &ai.CurrentOrder{Kind: kind}
	switch active.order_type {
	case OrderType_UnitMoveZone, OrderType_UnitAttackZone:
		if _, zone := invertZoneKey(uint(active.target_id), risq); zone != nil {
			zone_ref := toZoneRef(zone)
			order.TargetZone = &zone_ref
		}
	case OrderType_UnitGather:
		if _, zone := invertZoneKey(uint(active.target_id), risq); zone != nil && zone.resource != nil {
			resource_view := toResourceView(zone.resource)
			order.TargetResource = &resource_view
		}
	case OrderType_UnitBuild:
		if _, _, zone := invertBuildKey(uint(active.target_id), risq); zone != nil {
			zone_ref := toZoneRef(zone)
			order.TargetZone = &zone_ref
		}
	case OrderType_UnitAttackSpace:
		if space := invertSpaceKey(uint(active.target_id), risq); space != nil {
			coord := toCoordinate(space.coordinate)
			order.TargetSpace = &coord
		}
	case OrderType_UnitAttackUnit:
		if target := risq.units[uint64(active.target_id)]; target != nil {
			unit_view := toUnitViewShallow(target)
			order.TargetUnit = &unit_view
		}
	case OrderType_UnitAttackBuilding, OrderType_UnitGarrison, OrderType_UnitRepair:
		if target := risq.buildings[uint64(active.target_id)]; target != nil {
			building_view := toBuildingView(target, risq)
			order.TargetBuilding = &building_view
		}
	}
	return order
}

func unitBuilds(unit_id uint32) []ai.Producible {
	builds := make([]ai.Producible, 0)
	for _, p := range unitConfigs[unit_id].builds {
		if p.kind != ProducibleKind_BUILDING {
			continue
		}
		cost, _ := buildingProductionCost(p.id)
		builds = append(builds, ai.Producible{Kind: ai.ProducibleBuilding, ID: p.id, Cost: toCost(cost)})
	}
	return builds
}

// Doesn't populate CurrentOrder; used for target references inside another unit's CurrentOrder,
// where two units targeting each other would otherwise recurse forever.
func toUnitViewShallow(u *RisqUnit) ai.UnitView {
	kind := ai.UnitMilitary
	if isEconomicUnit(u.unit_id) {
		kind = ai.UnitEconomic
	}
	view := ai.UnitView{
		InternalID:     u.internal_id,
		UnitID:         u.unit_id,
		Kind:           kind,
		Location:       toZoneRef(u.zone),
		CurrentStamina: u.current_stamina,
		Builds:         unitBuilds(u.unit_id),
	}
	if u.garrisoned_in != nil {
		id := u.garrisoned_in.internal_id
		view.GarrisonedIn = &id
		view.Location = toZoneRef(u.garrisoned_in.zone)
	}
	return view
}

func toUnitView(u *RisqUnit, risq *GameRisq) ai.UnitView {
	view := toUnitViewShallow(u)
	view.CurrentOrder = currentOrder(u, risq)
	return view
}

func toResourceView(r *RisqResource) ai.ResourceView {
	return ai.ResourceView{
		InternalID: r.internal_id,
		Category:   toAiCategory(r.category()),
		Location:   toZoneRef(r.zone),
		AmountLeft: r.resources_left,
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
		GarrisonCount:     len(b.garrisoned_units),
		GarrisonCapacity:  int(b.garrison_capacity),
	}
}

func (v *aiView) unitView(u *RisqUnit) ai.UnitView {
	view := toUnitView(u, v.risq)
	if override, touched := v.assigned_units[u.internal_id]; touched {
		view.CurrentOrder = override
	}
	return view
}

func (v *aiView) assignUnit(id uint64, order *ai.CurrentOrder) {
	v.assigned_units[id] = order
}

func (v *aiView) claimBuilding(id uint64) {
	v.claimed_buildings[id] = true
}

func sortUnitViews(units []ai.UnitView) []ai.UnitView {
	sort.Slice(units, func(i, j int) bool { return units[i].InternalID < units[j].InternalID })
	return units
}

func sortBuildingViews(buildings []ai.BuildingView) []ai.BuildingView {
	sort.Slice(buildings, func(i, j int) bool { return buildings[i].InternalID < buildings[j].InternalID })
	return buildings
}

func (v *aiView) Units() []ai.UnitView {
	units := make([]ai.UnitView, 0, len(v.player.units))
	for _, u := range v.player.units {
		if u != nil && !u.deleted {
			units = append(units, v.unitView(u))
		}
	}
	return sortUnitViews(units)
}

func (v *aiView) IdleUnits() []ai.UnitView {
	return v.EligibleUnits()
}

func (v *aiView) EligibleUnits(kinds ...ai.OrderKind) []ai.UnitView {
	allowed := make(map[ai.OrderKind]bool, len(kinds))
	for _, k := range kinds {
		allowed[k] = true
	}
	units := make([]ai.UnitView, 0)
	for _, u := range v.player.units {
		if u == nil || u.deleted {
			continue
		}
		view := v.unitView(u)
		if view.CurrentOrder == nil || allowed[view.CurrentOrder.Kind] {
			units = append(units, view)
		}
	}
	return sortUnitViews(units)
}

func (v *aiView) Buildings() []ai.BuildingView {
	buildings := make([]ai.BuildingView, 0, len(v.player.buildings))
	for _, b := range v.player.buildings {
		if b != nil && !b.deleted {
			buildings = append(buildings, toBuildingView(b, v.risq))
		}
	}
	return sortBuildingViews(buildings)
}

func (v *aiView) IdleBuildings() []ai.BuildingView {
	buildings := make([]ai.BuildingView, 0)
	for _, b := range v.player.buildings {
		if b != nil && !b.deleted && buildingIdle(b) && !v.claimed_buildings[b.internal_id] {
			buildings = append(buildings, toBuildingView(b, v.risq))
		}
	}
	return sortBuildingViews(buildings)
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

func (v *aiView) TurnNumber() int {
	return int(v.risq.turn_number)
}

// True once, for every other non-eliminated player, at least one of their buildings has ever been seen
func (v *aiView) AllEnemiesFound() bool {
	discovered := make(map[int]bool)
	for _, row := range v.risq.spaces {
		for _, space := range row {
			for _, cache := range space.building_cache[v.playerId()] {
				if cache.player_id != v.playerId() {
					discovered[cache.player_id] = true
				}
			}
			if space.getVisibility(v.playerId()) >= VisibilityPoor {
				for _, b := range space.buildings {
					if b != nil && !b.deleted && b.player_id != v.playerId() {
						discovered[b.player_id] = true
					}
				}
			}
		}
	}
	for _, p := range v.risq.players {
		if p == v.player || p.eliminated {
			continue
		}
		if !discovered[p.player.Player_id] {
			return false
		}
	}
	return true
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
				units = append(units, toUnitView(u, v.risq))
			}
		}
	}
	return sortUnitViews(units)
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
	return sortBuildingViews(buildings)
}

func (v *aiView) NearestResource(from ai.ZoneRef, category ai.ResourceCategory) (ai.ResourceView, bool) {
	from_space, from_zone := v.resolveZone(from)
	if from_space == nil {
		return ai.ResourceView{}, false
	}
	var tied []*RisqZone
	best_distance := -1
	for _, row := range v.risq.spaces {
		for _, space := range row {
			if space.getVisibility(v.playerId()) < VisibilityFog {
				continue
			}
			distance := int(game_utils.AxialDistance(from_space.coordinate, space.coordinate) * 6)
			for _, zone_row := range space.zones {
				for _, zone := range zone_row {
					if zone.resource == nil || zone.resource.resources_left <= 0 || zone.resource.category() != RisqResourceCategory(category) {
						continue
					}
					d := distance
					if space == from_space {
						d = zoneDistanceWithinSpace(from_zone, zone)
					}
					if best_distance == -1 || d < best_distance {
						tied, best_distance = []*RisqZone{zone}, d
					} else if d == best_distance {
						tied = append(tied, zone)
					}
				}
			}
		}
	}
	if len(tied) == 0 {
		return ai.ResourceView{}, false
	}
	return toResourceView(tied[v.player.rng.Intn(len(tied))].resource), true
}

func (v *aiView) BuildCost(building_id uint32) ai.Cost {
	cost, _ := buildingProductionCost(building_id)
	return toCost(cost)
}

func (v *aiView) UnitCost(unit_id uint32) ai.Cost {
	cost, _ := unitProductionCost(unit_id)
	return toCost(cost)
}

func (v *aiView) TechCost(tech_id uint32) ai.Cost {
	return toCost(techConfigs[tech_id].cost)
}

func (v *aiView) NearestUnexplored(from ai.ZoneRef) ([]ai.ZoneRef, bool) {
	from_space, _ := v.resolveZone(from)
	if from_space == nil {
		return nil, false
	}
	var tied []*RisqSpace
	best_distance := -1
	for _, row := range v.risq.spaces {
		for _, space := range row {
			if space.getVisibility(v.playerId()) != VisibilityUnexplored {
				continue
			}
			d := int(game_utils.AxialDistance(from_space.coordinate, space.coordinate))
			if best_distance == -1 || d < best_distance {
				tied, best_distance = []*RisqSpace{space}, d
			} else if d == best_distance {
				tied = append(tied, space)
			}
		}
	}
	if len(tied) == 0 {
		return nil, false
	}
	zones := make([]ai.ZoneRef, len(tied))
	for i, space := range tied {
		zones[i] = toZoneRef(space.getCenterZone())
	}
	return zones, true
}

func (v *aiView) NearestBuildSite(from ai.ZoneRef, building_id uint32) (ai.ZoneRef, bool) {
	from_space, from_zone := v.resolveZone(from)
	if from_space == nil {
		return ai.ZoneRef{}, false
	}
	var tied []*RisqZone
	best_distance := -1
	for _, row := range v.risq.spaces {
		for _, space := range row {
			if !space.buildableBy(v.playerId()) {
				continue
			}
			distance := int(game_utils.AxialDistance(from_space.coordinate, space.coordinate) * 6)
			for _, zone_row := range space.zones {
				for _, zone := range zone_row {
					if zone.resource != nil || zone.building != nil {
						continue
					}
					d := distance
					if space == from_space {
						d = zoneDistanceWithinSpace(from_zone, zone)
					}
					if best_distance == -1 || d < best_distance {
						tied, best_distance = []*RisqZone{zone}, d
					} else if d == best_distance {
						tied = append(tied, zone)
					}
				}
			}
		}
	}
	if len(tied) == 0 {
		return ai.ZoneRef{}, false
	}
	return toZoneRef(tied[v.player.rng.Intn(len(tied))]), true
}

func (v *aiView) MoveOrder(u ai.UnitView, target ai.ZoneRef, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindMove, TargetZone: &target})
	_, zone := v.resolveZone(target)
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: uint8(OrderType_UnitMoveZone), TargetID: int64(zone.coordinate_key), ClearPreviousOrders: clear_previous}
}

func (v *aiView) GatherOrder(u ai.UnitView, target ai.ResourceView, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindGather, TargetResource: &target})
	_, zone := v.resolveZone(target.Location)
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: uint8(OrderType_UnitGather), TargetID: int64(zone.coordinate_key), ClearPreviousOrders: clear_previous}
}

func (v *aiView) BuildOrder(u ai.UnitView, building_id uint32, target ai.ZoneRef, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindBuild, TargetZone: &target})
	_, zone := v.resolveZone(target)
	target_id := util.Pair(int(building_id), int(zone.coordinate_key))
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: uint8(OrderType_UnitBuild), TargetID: int64(target_id), ClearPreviousOrders: clear_previous}
}

func (v *aiView) RepairOrder(u ai.UnitView, target ai.BuildingView, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindRepair, TargetBuilding: &target})
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: uint8(OrderType_UnitRepair), TargetID: int64(target.InternalID), ClearPreviousOrders: clear_previous}
}

func (v *aiView) AttackUnitOrder(u ai.UnitView, target ai.UnitView, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindAttackUnit, TargetUnit: &target})
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: uint8(OrderType_UnitAttackUnit), TargetID: int64(target.InternalID), ClearPreviousOrders: clear_previous}
}

func (v *aiView) AttackBuildingOrder(u ai.UnitView, target ai.BuildingView, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindAttackBuilding, TargetBuilding: &target})
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: uint8(OrderType_UnitAttackBuilding), TargetID: int64(target.InternalID), ClearPreviousOrders: clear_previous}
}

func (v *aiView) AttackSpaceOrder(u ai.UnitView, target ai.Coordinate, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindAttackSpace, TargetSpace: &target})
	key := util.Pair(target.X, target.Y)
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: uint8(OrderType_UnitAttackSpace), TargetID: int64(key), ClearPreviousOrders: clear_previous}
}

func (v *aiView) AttackZoneOrder(u ai.UnitView, target ai.ZoneRef, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindAttackZone, TargetZone: &target})
	_, zone := v.resolveZone(target)
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: uint8(OrderType_UnitAttackZone), TargetID: int64(zone.coordinate_key), ClearPreviousOrders: clear_previous}
}

func (v *aiView) GarrisonOrder(u ai.UnitView, target ai.BuildingView, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindGarrison, TargetBuilding: &target})
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: uint8(OrderType_UnitGarrison), TargetID: int64(target.InternalID), ClearPreviousOrders: clear_previous}
}

func (v *aiView) UngarrisonOrder(u ai.UnitView, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindUngarrison})
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: uint8(OrderType_UnitUngarrison), ClearPreviousOrders: clear_previous}
}

func (v *aiView) DeleteUnitOrder(u ai.UnitView) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindDelete})
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
