package aibridge

import (
	"sort"

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

func unitBuilds(unit_id uint32) []ai.Producible {
	builds := make([]ai.Producible, 0)
	for _, p := range defs.UnitConfigs[unit_id].Builds {
		if p.Kind != defs.ProducibleKind_BUILDING {
			continue
		}
		cost, _ := defs.BuildingProductionCost(p.Id)
		builds = append(builds, ai.Producible{Kind: ai.ProducibleBuilding, ID: p.Id, Cost: toCost(cost)})
	}
	return builds
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
