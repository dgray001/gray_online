package aibridge

import (
	"sort"
	"sync"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
)

func toCoordinate(c snapCoordinate) ai.Coordinate {
	return ai.Coordinate{X: c.X, Y: c.Y}
}

func zoneRefOf(space snapCoordinate, zone snapCoordinate) ai.ZoneRef {
	return ai.ZoneRef{Space: toCoordinate(space), Zone: toCoordinate(zone)}
}

func spaceKey(c ai.Coordinate) uint {
	return util.Pair(c.X, c.Y)
}

func zoneKey(ref ai.ZoneRef) uint {
	return util.Pair(int(spaceKey(ref.Space)), int(util.Pair(ref.Zone.X, ref.Zone.Y)))
}

func zoneRefFromKey(k uint) ai.ZoneRef {
	space_key, local_key := util.InvertPair(k)
	x, y := util.InvertPair(uint(space_key))
	i, j := util.InvertPair(uint(local_key))
	return ai.ZoneRef{Space: ai.Coordinate{X: x, Y: y}, Zone: ai.Coordinate{X: i, Y: j}}
}

func axialDistance(a ai.Coordinate, b ai.Coordinate) int {
	return int(game_utils.AxialDistance(game_utils.Coordinate2D{X: a.X, Y: a.Y}, game_utils.Coordinate2D{X: b.X, Y: b.Y}))
}

func spaceOwner(space *snapSpace) (int, bool) {
	if space == nil || space.Ownership == nil {
		return -1, false
	}
	return *space.Ownership, true
}

func toCost(c defs.RisqResourceCost) ai.Cost {
	return ai.Cost{Food: c.Food, Wood: c.Wood, Stone: c.Stone, Gold: c.Gold}
}

func toAiCategory(c defs.RisqResourceCategory) ai.ResourceCategory {
	switch c {
	case defs.RisqResourceCategory_FOOD:
		return ai.ResourceFood
	case defs.RisqResourceCategory_WOOD:
		return ai.ResourceWood
	case defs.RisqResourceCategory_STONE:
		return ai.ResourceStone
	default:
		return ai.ResourceGold
	}
}

func toOrderKind(order_type defs.OrderType) (ai.OrderKind, bool) {
	switch order_type {
	case defs.OrderType_UnitMoveSpace, defs.OrderType_UnitMoveZone:
		return ai.OrderKindMove, true
	case defs.OrderType_UnitGather:
		return ai.OrderKindGather, true
	case defs.OrderType_UnitBuild:
		return ai.OrderKindBuild, true
	case defs.OrderType_UnitRepair:
		return ai.OrderKindRepair, true
	case defs.OrderType_UnitAttackSpace:
		return ai.OrderKindAttackSpace, true
	case defs.OrderType_UnitAttackZone:
		return ai.OrderKindAttackZone, true
	case defs.OrderType_UnitAttackUnit:
		return ai.OrderKindAttackUnit, true
	case defs.OrderType_UnitAutoAttackUnit:
		return ai.OrderKindAutoAttackUnit, true
	case defs.OrderType_UnitAutoAttackBuilding:
		return ai.OrderKindAutoAttackBuilding, true
	case defs.OrderType_UnitAttackBuilding:
		return ai.OrderKindAttackBuilding, true
	case defs.OrderType_UnitGarrison:
		return ai.OrderKindGarrison, true
	case defs.OrderType_UnitUngarrison:
		return ai.OrderKindUngarrison, true
	case defs.OrderType_UnitDelete:
		return ai.OrderKindDelete, true
	case defs.OrderType_UnitRenew:
		return ai.OrderKindRenew, true
	default:
		return 0, false
	}
}

func toBuildingOrderKind(order_type defs.OrderType) (ai.BuildingOrderKind, bool) {
	switch order_type {
	case defs.OrderType_BuildingCreate:
		return ai.BuildingOrderCreate, true
	case defs.OrderType_BuildingResearch:
		return ai.BuildingOrderResearch, true
	case defs.OrderType_BuildingDelete:
		return ai.BuildingOrderDelete, true
	case defs.OrderType_BuildingAttackUnit:
		return ai.BuildingOrderAttackUnit, true
	case defs.OrderType_BuildingAttackBuilding:
		return ai.BuildingOrderAttackBuilding, true
	}
	return 0, false
}

func toAiTargetCategories(priority []int) []ai.TargetCategory {
	categories := make([]ai.TargetCategory, len(priority))
	for i, c := range priority {
		categories[i] = ai.TargetCategory(c)
	}
	return categories
}

var unitBuildsByConfig sync.Map // *defs.Config -> map[uint32][]ai.Producible, shared read-only

func buildUnitBuilds(config *defs.Config) map[uint32][]ai.Producible {
	table := make(map[uint32][]ai.Producible, len(config.Units))
	for unit_id, unit := range config.Units {
		builds := make([]ai.Producible, 0)
		for _, p := range unit.Builds {
			if p.Kind != defs.ProducibleKind_BUILDING {
				continue
			}
			cost, _ := config.BuildingProductionCost(p.Id)
			builds = append(builds, ai.Producible{Kind: ai.ProducibleBuilding, ID: p.Id, Cost: toCost(cost), StaminaCost: config.Buildings[p.Id].Build_stamina})
		}
		table[unit_id] = builds
	}
	return table
}

func (v *aiView) unitBuilds(unit_id uint32) []ai.Producible {
	table, ok := unitBuildsByConfig.Load(v.config)
	if !ok {
		table, _ = unitBuildsByConfig.LoadOrStore(v.config, buildUnitBuilds(v.config))
	}
	return table.(map[uint32][]ai.Producible)[unit_id]
}

func sortUnitViews(units []ai.UnitView) []ai.UnitView {
	sort.Slice(units, func(i, j int) bool { return units[i].InternalID < units[j].InternalID })
	return units
}

func sortBuildingViews(buildings []ai.BuildingView) []ai.BuildingView {
	sort.Slice(buildings, func(i, j int) bool { return buildings[i].InternalID < buildings[j].InternalID })
	return buildings
}

func zoneRefLess(a ai.ZoneRef, b ai.ZoneRef) bool {
	ka := [4]int{a.Space.X, a.Space.Y, a.Zone.X, a.Zone.Y}
	kb := [4]int{b.Space.X, b.Space.Y, b.Zone.X, b.Zone.Y}
	for i := range ka {
		if ka[i] != kb[i] {
			return ka[i] < kb[i]
		}
	}
	return false
}

func (v *aiView) currentOrder(orders []snapOrder) *ai.CurrentOrder {
	if len(orders) == 0 {
		return nil
	}
	active := orders[0]
	kind, ok := toOrderKind(active.OrderType)
	if !ok {
		return nil
	}
	order := &ai.CurrentOrder{Kind: kind}
	switch active.OrderType {
	case defs.OrderType_UnitMoveZone, defs.OrderType_UnitAttackZone:
		ref := zoneRefFromKey(uint(active.TargetId))
		order.TargetZone = &ref
	case defs.OrderType_UnitGather:
		if resource, ok := v.gatherTargetAt(zoneRefFromKey(uint(active.TargetId))); ok {
			order.TargetResource = &resource
		}
	case defs.OrderType_UnitBuild:
		_, zone_key := util.InvertPair(uint(active.TargetId))
		ref := zoneRefFromKey(uint(zone_key))
		order.TargetZone = &ref
	case defs.OrderType_UnitMoveSpace, defs.OrderType_UnitAttackSpace:
		x, y := util.InvertPair(uint(active.TargetId))
		order.TargetSpace = &ai.Coordinate{X: x, Y: y}
	case defs.OrderType_UnitAttackUnit, defs.OrderType_UnitAutoAttackUnit:
		if target, ok := v.units[uint64(active.TargetId)]; ok {
			view := v.unitViewShallow(target)
			order.TargetUnit = &view
		}
	case defs.OrderType_UnitAttackBuilding, defs.OrderType_UnitAutoAttackBuilding, defs.OrderType_UnitGarrison, defs.OrderType_UnitRepair, defs.OrderType_UnitRenew:
		if target, ok := v.buildings[uint64(active.TargetId)]; ok {
			view := v.buildingView(target)
			order.TargetBuilding = &view
		}
	}
	return order
}

func (v *aiView) unitLocation(u *snapUnit) ai.ZoneRef {
	if u.GarrisonedIn != nil {
		if b, ok := v.buildings[*u.GarrisonedIn]; ok {
			return zoneRefOf(b.Space, b.Zone)
		}
	}
	return zoneRefOf(u.Space, u.Zone)
}

// Doesn't populate CurrentOrder, so two units targeting each other can't recurse forever
func (v *aiView) unitViewShallow(u *snapUnit) ai.UnitView {
	kind := ai.UnitMilitary
	if u.UnitType == defs.UnitType_ECONOMIC {
		kind = ai.UnitEconomic
	}
	return ai.UnitView{
		InternalID:       u.InternalId,
		PlayerID:         u.PlayerId,
		UnitID:           u.UnitId,
		Type:             ai.UnitType(u.UnitType),
		Kind:             kind,
		Location:         v.unitLocation(u),
		CurrentStamina:   u.CurrentStamina,
		MaxStamina:       u.MaxStamina,
		AttackRange:      ai.AttackRange(v.config.Units[u.UnitId].Attack_range),
		CanAttack:        v.config.Units[u.UnitId].Attack_type != defs.AttackType_NONE,
		Builds:           v.unitBuilds(u.UnitId),
		GarrisonedIn:     u.GarrisonedIn,
		Stance:           ai.UnitStance(u.Stance),
		InterruptCurrent: u.InterruptCurrent,
		AttackBack:       u.AttackBack,
		TargetPriority:   toAiTargetCategories(u.TargetPriority),
		Health:           u.CombatStats.Health,
		MaxHealth:        float64(u.CombatStats.MaxHealth),
	}
}

// Orders minus any this planning pass already emitted a cancel for
func (v *aiView) liveOrders(orders []snapOrder) []snapOrder {
	live := make([]snapOrder, 0, len(orders))
	for _, o := range orders {
		if !v.cancelled_orders[o.InternalId] {
			live = append(live, o)
		}
	}
	return live
}

func (v *aiView) unitView(u *snapUnit) ai.UnitView {
	view := v.unitViewShallow(u)
	live := v.liveOrders(u.ActiveOrders)
	view.CurrentOrder = v.currentOrder(live)
	if override, touched := v.assigned_units[u.InternalId]; touched {
		view.CurrentOrder = override
	}
	for _, o := range live {
		if kind, ok := toOrderKind(o.OrderType); ok && v.order_ids[o.InternalId] {
			view.ActiveOrders = append(view.ActiveOrders, ai.ActiveUnitOrder{ID: o.InternalId, Kind: kind})
		}
	}
	return view
}

func (v *aiView) producibles(b *snapBuilding) []ai.Producible {
	producibles := make([]ai.Producible, 0)
	for _, p := range v.config.Buildings[b.BuildingId].Produces {
		switch p.Kind {
		case defs.ProducibleKind_UNIT:
			if !v.techMet(v.config.Units[p.Id].Required_tech_id) {
				continue
			}
			cost, _ := v.config.UnitProductionCost(p.Id)
			producibles = append(producibles, ai.Producible{Kind: ai.ProducibleUnit, ID: p.Id, Cost: toCost(cost), UnitType: ai.UnitType(v.config.Units[p.Id].Unit_type), StaminaCost: v.config.Units[p.Id].Production_stamina})
		case defs.ProducibleKind_TECH:
			if _, researching_or_done := v.me.ResearchedTechs[p.Id]; researching_or_done || v.planned_techs[p.Id] || !v.techMet(v.config.Techs[p.Id].Required_tech_id) {
				continue
			}
			producibles = append(producibles, ai.Producible{Kind: ai.ProducibleTech, ID: p.Id, Cost: toCost(v.config.Techs[p.Id].Cost), StaminaCost: v.config.Techs[p.Id].Research_stamina})
		}
	}
	return producibles
}

func (v *aiView) AvailableMercenaries() []ai.Producible {
	mercenaries := make([]ai.Producible, 0, len(v.me.AvailableMercenaries))
	for _, mercenary := range v.me.AvailableMercenaries {
		mercenaries = append(mercenaries, ai.Producible{
			Kind: ai.ProducibleKind(mercenary.Kind), ID: mercenary.ID,
			UnitType: ai.UnitType(mercenary.UnitType), StaminaCost: mercenary.Stamina,
			Cost: ai.Cost{Food: mercenary.Cost.Food, Wood: mercenary.Cost.Wood, Stone: mercenary.Cost.Stone, Gold: mercenary.Cost.Gold},
		})
	}
	return mercenaries
}

func (v *aiView) buildingView(b *snapBuilding) ai.BuildingView {
	config := v.config.Buildings[b.BuildingId]
	return ai.BuildingView{
		InternalID:        b.InternalId,
		PlayerID:          b.PlayerId,
		BuildingID:        b.BuildingId,
		Location:          zoneRefOf(b.Space, b.Zone),
		AttackRange:       ai.AttackRange(config.Attack_range),
		MaxStamina:        b.MaxStamina,
		Visible:           true,
		UnderConstruction: b.UnderConstruction,
		Idle:              !b.UnderConstruction && len(b.ActiveOrders) == 0 && len(config.Produces) > 0,
		Producibles:       v.producibles(b),
		GarrisonCount:     len(b.GarrisonedUnits),
		GarrisonCapacity:  b.GarrisonCapacity,
		Gatherable:        config.IsGatherable(),
		ResourceCategory:  toAiCategory(config.Gather.Resource_category),
		ResourcesLeft:     b.ResourcesLeft,
		GatherSpeed:       config.Gather.Base_gather_speed,
		GatherCapacity:    config.Gather.Gather_capacity,
		Renewing:          b.Renewing,
		RenewCost:         toCost(config.Gather.Renew_cost),
		Health:            b.CombatStats.Health,
		MaxHealth:         float64(b.CombatStats.MaxHealth),
		CanAttack:         config.Attack_type != defs.AttackType_NONE,
	}
}

func (v *aiView) ownBuildingView(b *snapBuilding) ai.BuildingView {
	view := v.buildingView(b)
	view.AutoAttack = b.AutoAttack
	view.InterruptCurrent = b.InterruptCurrent
	view.TargetPriority = toAiTargetCategories(b.TargetPriority)
	live := v.liveOrders(b.ActiveOrders)
	for _, o := range live {
		if kind, ok := toBuildingOrderKind(o.OrderType); ok && v.order_ids[o.InternalId] {
			view.ActiveOrders = append(view.ActiveOrders, ai.ActiveBuildingOrder{ID: o.InternalId, Kind: kind, ItemID: uint32(o.TargetId)})
		}
	}
	view.PlannedProduction = v.planned_production[b.InternalId]
	view.Idle = !b.UnderConstruction && len(live) == 0 && view.PlannedProduction == 0 && len(v.config.Buildings[b.BuildingId].Produces) > 0
	return view
}
