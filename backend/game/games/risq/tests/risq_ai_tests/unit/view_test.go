package unit

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/util"
)

type fakeView struct {
	calls                        []orderCall
	stateful                     bool
	plannedTechs                 map[uint32]bool
	assignedUnits                map[uint64]*ai.CurrentOrder
	queuedJobs                   map[uint64]*ai.CurrentOrder
	allowed                      map[string]bool
	queries                      []string
	matchingSpaces               func(ai.SpaceCondition) []ai.Coordinate
	techQueries                  int
	spaceDistances               map[[2]ai.Coordinate]int
	locationDistances            map[[2]ai.ZoneRef]int
	nickname                     string
	units                        []ai.UnitView
	buildings                    []ai.BuildingView
	resources                    map[ai.ResourceCategory]float64
	popCurrent, popLimit, popMax int
	foundations                  []ai.FoundationView

	visibleEnemyUnits     []ai.UnitView
	visibleEnemyBuildings []ai.BuildingView
	knownEnemyBuildings   []ai.BuildingView

	turnNumber, mapSize, numPlayers int
	allSpaces                       []ai.SpaceInfo
	allRegions                      []ai.RegionInfo
	availableMercenaries            []ai.Producible
	allZones                        []ai.ZoneInfo
	playerIDs                       []int
	playerID                        int
	playerScores                    map[int]int

	techResearched    map[uint32]bool
	buildingAvailable map[uint32]bool
	buildCost         map[uint32]ai.Cost
	unitCost          map[uint32]ai.Cost
	techCost          map[uint32]ai.Cost

	knownResources []ai.ResourceView
}

type orderCall struct {
	Kind    uint8
	Subject uint64
	Target  uint64
	Item    uint32
	Place   ai.ZoneRef
	Clear   bool
}

func assertCalls(t *testing.T, decision ai.Decision, view *fakeView, calls ...orderCall) {
	t.Helper()
	if len(view.calls) != len(calls) || (len(calls) > 0 && !reflect.DeepEqual(view.calls, calls)) {
		t.Fatalf("calls = %+v, want %+v", view.calls, calls)
	}
	if len(decision.Orders) != len(calls) {
		t.Fatalf("orders = %+v, want %d", decision.Orders, len(calls))
	}
}

func (f *fakeView) query(name string) {
	if f.allowed == nil {
		return
	}
	if !f.allowed[name] {
		panic("unconfigured View." + name)
	}
	f.queries = append(f.queries, name)
}

func (f *fakeView) record(call orderCall) ai.Order {
	f.calls = append(f.calls, call)
	if f.stateful {
		f.bookkeep(call)
	}
	return ai.Order{Subjects: []uint64{call.Subject}, OrderType: call.Kind, TargetID: int64(len(f.calls)), ClearPreviousOrders: call.Clear}
}

func (f *fakeView) bookkeep(call orderCall) {
	if call.Kind == 20 {
		f.cancelLiveOrder(call.Target)
	}
	if call.Kind == 3 && !slices.ContainsFunc(f.foundations, func(foundation ai.FoundationView) bool { return foundation.Location == call.Place }) {
		f.foundations = append(f.foundations, ai.FoundationView{BuildingID: call.Item, Location: call.Place, Planned: true})
	}
	if call.Kind == 18 {
		f.foundations = slices.DeleteFunc(slices.Clone(f.foundations), func(foundation ai.FoundationView) bool { return foundation.Location == call.Place })
	}
	if call.Kind == 14 {
		if f.plannedTechs == nil {
			f.plannedTechs = map[uint32]bool{}
		}
		f.plannedTechs[call.Item] = true
		for index := range f.buildings {
			f.buildings[index].Producibles = slices.DeleteFunc(slices.Clone(f.buildings[index].Producibles), func(item ai.Producible) bool { return item.Kind == ai.ProducibleTech && item.ID == call.Item })
		}
	}
	kinds := map[uint8]ai.OrderKind{1: ai.OrderKindMove, 2: ai.OrderKindGather, 3: ai.OrderKindBuild, 4: ai.OrderKindRepair, 5: ai.OrderKindRenew, 6: ai.OrderKindAttackUnit, 7: ai.OrderKindAttackBuilding, 8: ai.OrderKindAttackSpace, 9: ai.OrderKindAttackZone, 10: ai.OrderKindGarrison, 11: ai.OrderKindUngarrison, 12: ai.OrderKindDelete}
	if kind, ok := kinds[call.Kind]; ok {
		order := f.currentOrder(call, kind)
		if f.assignedUnits == nil {
			f.assignedUnits = map[uint64]*ai.CurrentOrder{}
		}
		f.assignedUnits[call.Subject] = order
		for index := range f.units {
			if f.units[index].InternalID == call.Subject {
				f.units[index].CurrentOrder = order
			}
		}
	}
	for index := range f.buildings {
		if f.buildings[index].InternalID == call.Subject && call.Kind >= 13 && call.Kind <= 17 {
			f.buildings[index].Idle = false
			if call.Kind == 13 || call.Kind == 14 {
				f.buildings[index].PlannedProduction++
			}
		}
	}
}

func (f *fakeView) currentOrder(call orderCall, kind ai.OrderKind) *ai.CurrentOrder {
	order := &ai.CurrentOrder{Kind: kind}
	switch call.Kind {
	case 1, 3, 9:
		order.TargetZone = &call.Place
	case 8:
		order.TargetSpace = &call.Place.Space
	case 2:
		for _, resource := range f.knownResources {
			if resource.InternalID == call.Target {
				order.TargetResource = &resource
				break
			}
		}
	case 4, 5, 7, 10:
		for _, building := range append(slices.Clone(f.buildings), f.knownEnemyBuildings...) {
			if building.InternalID == call.Target {
				order.TargetBuilding = &building
				break
			}
		}
	case 6:
		for _, unit := range f.visibleEnemyUnits {
			if unit.InternalID == call.Target {
				order.TargetUnit = &unit
				break
			}
		}
	}
	return order
}

func (f *fakeView) Logger() *util.Logger        { return nil }
func (f *fakeView) Revision() uint64            { return 0 }
func (f *fakeView) OrderQueue() []ai.OrderEntry { return nil }
func (f *fakeView) Nickname() string            { return f.nickname }
func (f *fakeView) Units() []ai.UnitView        { f.query("Units"); return slices.Clone(f.units) }
func (f *fakeView) IdleUnits() []ai.UnitView {
	f.query("IdleUnits")
	return slices.DeleteFunc(slices.Clone(f.units), func(unit ai.UnitView) bool { return unit.CurrentOrder != nil })
}
func (f *fakeView) Buildings() []ai.BuildingView {
	f.query("Buildings")
	return slices.Clone(f.buildings)
}
func (f *fakeView) IdleBuildings() []ai.BuildingView {
	f.query("IdleBuildings")
	return slices.DeleteFunc(slices.Clone(f.buildings), func(building ai.BuildingView) bool { return !building.Idle })
}
func (f *fakeView) Resource(category ai.ResourceCategory) float64 {
	f.query("Resource")
	return f.resources[category]
}
func (f *fakeView) Population() (current, limit int) {
	f.query("Population")
	return f.popCurrent, f.popLimit
}
func (f *fakeView) MaxPopulation() int { f.query("MaxPopulation"); return f.popMax }
func (f *fakeView) Foundations() []ai.FoundationView {
	f.query("Foundations")
	foundations := slices.Clone(f.foundations)
	for index := range foundations {
		for _, job := range f.assignedUnits {
			if job.Kind == ai.OrderKindBuild && job.TargetZone != nil && *job.TargetZone == foundations[index].Location {
				foundations[index].Builders++
			}
		}
	}
	return foundations
}

func (f *fakeView) VisibleEnemyUnits() []ai.UnitView {
	f.query("VisibleEnemyUnits")
	return slices.Clone(f.visibleEnemyUnits)
}
func (f *fakeView) VisibleEnemyBuildings() []ai.BuildingView {
	f.query("VisibleEnemyBuildings")
	if f.visibleEnemyBuildings != nil {
		return slices.Clone(f.visibleEnemyBuildings)
	}
	return slices.Clone(f.knownEnemyBuildings)
}
func (f *fakeView) KnownEnemyBuildings() []ai.BuildingView {
	f.query("KnownEnemyBuildings")
	return slices.Clone(f.knownEnemyBuildings)
}

func (f *fakeView) KnownResources(category ai.ResourceCategory) []ai.ResourceView {
	f.query("KnownResources")
	var res []ai.ResourceView
	for _, r := range f.knownResources {
		if r.Category == category {
			res = append(res, r)
		}
	}
	return res
}
func (f *fakeView) TurnNumber() int           { return f.turnNumber }
func (f *fakeView) MapSize() int              { return f.mapSize }
func (f *fakeView) NumPlayers() int           { return f.numPlayers }
func (f *fakeView) AllSpaces() []ai.SpaceInfo { f.query("AllSpaces"); return slices.Clone(f.allSpaces) }
func (f *fakeView) AllRegions() []ai.RegionInfo {
	f.query("AllRegions")
	return slices.Clone(f.allRegions)
}
func (f *fakeView) AvailableMercenaries() []ai.Producible {
	return slices.Clone(f.availableMercenaries)
}
func (f *fakeView) AllZones() []ai.ZoneInfo             { f.query("AllZones"); return slices.Clone(f.allZones) }
func (f *fakeView) PlayerIDs() []int                    { return f.playerIDs }
func (f *fakeView) PlayerID() int                       { return f.playerID }
func (f *fakeView) PlayerScore(player_id int) int       { return f.playerScores[player_id] }
func (f *fakeView) PlayerEliminated(player_id int) bool { return false }
func (f *fakeView) MatchingSpaces(condition ai.SpaceCondition) []ai.Coordinate {
	f.query("MatchingSpaces")
	if f.matchingSpaces != nil {
		return slices.Clone(f.matchingSpaces(condition))
	}
	return nil
}
func (f *fakeView) SpaceDistance(a ai.Coordinate, b ai.Coordinate) int {
	f.query("SpaceDistance")
	if f.spaceDistances == nil {
		return 0
	}
	distance, ok := f.spaceDistances[[2]ai.Coordinate{a, b}]
	if !ok {
		panic("unconfigured SpaceDistance")
	}
	return distance
}
func (f *fakeView) SpaceNeighbors(space ai.Coordinate) []ai.Coordinate {
	f.query("SpaceNeighbors")
	return nil
}
func (f *fakeView) LocationDistance(a ai.ZoneRef, b ai.ZoneRef) int {
	f.query("LocationDistance")
	if f.locationDistances == nil {
		return 0
	}
	distance, ok := f.locationDistances[[2]ai.ZoneRef{a, b}]
	if !ok {
		panic("unconfigured LocationDistance")
	}
	return distance
}
func (f *fakeView) TechResearched(tech_id uint32) bool {
	f.techQueries++
	return f.techResearched[tech_id]
}
func (f *fakeView) BuildingAvailable(building_id uint32) bool {
	return f.buildingAvailable[building_id]
}
func (f *fakeView) ValidBuildZone(uint32, ai.ZoneRef) bool { return true }
func (f *fakeView) BuildCost(building_id uint32) ai.Cost   { return f.buildCost[building_id] }
func (f *fakeView) UnitCost(unit_id uint32) ai.Cost        { return f.unitCost[unit_id] }
func (f *fakeView) TechCost(tech_id uint32) ai.Cost        { return f.techCost[tech_id] }

var fakeConfig = map[string]float64{"unit.cost_food.2": 3, "unit.cost_wood.2": 4, "building.max_health.7": 90}

func (f *fakeView) ConfigValue(table string, field string, id uint32) (float64, error) {
	return fakeConfig[fmt.Sprintf("%s.%s.%d", table, field, id)], nil
}

func (f *fakeView) MoveOrder(u ai.UnitView, target ai.ZoneRef, clear_previous bool) ai.Order {
	return f.record(orderCall{Kind: 1, Subject: u.InternalID, Place: target, Clear: clear_previous})
}
func (f *fakeView) AllUnits() []ai.UnitView {
	return slices.Concat(f.Units(), f.VisibleEnemyUnits())
}
func (f *fakeView) AllBuildings() []ai.BuildingView {
	return slices.Concat(f.Buildings(), f.KnownEnemyBuildings())
}
func (f *fakeView) Issue(r ai.IssueRequest) ai.Order { return ai.DispatchIssue(f, r) }
func (f *fakeView) MoveSpaceOrder(u ai.UnitView, target ai.Coordinate, clear_previous bool) ai.Order {
	return f.record(orderCall{Kind: 21, Subject: u.InternalID, Place: ai.ZoneRef{Space: target}, Clear: clear_previous})
}
func (f *fakeView) GatherOrder(u ai.UnitView, target ai.ResourceView, clear_previous bool) ai.Order {
	return f.record(orderCall{Kind: 2, Subject: u.InternalID, Target: target.InternalID, Place: target.Location, Clear: clear_previous})
}
func (f *fakeView) BuildOrder(u ai.UnitView, building_id uint32, target ai.ZoneRef, clear_previous bool) ai.Order {
	return f.record(orderCall{Kind: 3, Subject: u.InternalID, Item: building_id, Place: target, Clear: clear_previous})
}
func (f *fakeView) RepairOrder(u ai.UnitView, target ai.BuildingView, clear_previous bool) ai.Order {
	return f.record(orderCall{Kind: 4, Subject: u.InternalID, Target: target.InternalID, Place: target.Location, Clear: clear_previous})
}
func (f *fakeView) RenewOrder(u ai.UnitView, target ai.BuildingView, clear_previous bool) ai.Order {
	return f.record(orderCall{Kind: 5, Subject: u.InternalID, Target: target.InternalID, Place: target.Location, Clear: clear_previous})
}
func (f *fakeView) AttackUnitOrder(u ai.UnitView, target ai.UnitView, clear_previous bool) ai.Order {
	return f.record(orderCall{Kind: 6, Subject: u.InternalID, Target: target.InternalID, Place: target.Location, Clear: clear_previous})
}
func (f *fakeView) AttackBuildingOrder(u ai.UnitView, target ai.BuildingView, clear_previous bool) ai.Order {
	return f.record(orderCall{Kind: 7, Subject: u.InternalID, Target: target.InternalID, Place: target.Location, Clear: clear_previous})
}
func (f *fakeView) AttackSpaceOrder(u ai.UnitView, target ai.Coordinate, clear_previous bool) ai.Order {
	return f.record(orderCall{Kind: 8, Subject: u.InternalID, Place: ai.ZoneRef{Space: target}, Clear: clear_previous})
}
func (f *fakeView) AttackZoneOrder(u ai.UnitView, target ai.ZoneRef, clear_previous bool) ai.Order {
	return f.record(orderCall{Kind: 9, Subject: u.InternalID, Place: target, Clear: clear_previous})
}
func (f *fakeView) GarrisonOrder(u ai.UnitView, target ai.BuildingView, clear_previous bool) ai.Order {
	return f.record(orderCall{Kind: 10, Subject: u.InternalID, Target: target.InternalID, Place: target.Location, Clear: clear_previous})
}
func (f *fakeView) UngarrisonOrder(u ai.UnitView, clear_previous bool) ai.Order {
	return f.record(orderCall{Kind: 11, Subject: u.InternalID, Clear: clear_previous})
}
func (f *fakeView) DeleteUnitOrder(u ai.UnitView) ai.Order {
	return f.record(orderCall{Kind: 12, Subject: u.InternalID, Clear: true})
}
func (f *fakeView) CreateUnitOrder(b ai.BuildingView, unit_id uint32) ai.Order {
	return f.record(orderCall{Kind: 13, Subject: b.InternalID, Item: unit_id})
}
func (f *fakeView) ResearchOrder(b ai.BuildingView, tech_id uint32) ai.Order {
	return f.record(orderCall{Kind: 14, Subject: b.InternalID, Item: tech_id})
}
func (f *fakeView) DeleteBuildingOrder(b ai.BuildingView) ai.Order {
	return f.record(orderCall{Kind: 15, Subject: b.InternalID})
}
func (f *fakeView) BuildingAttackUnitOrder(b ai.BuildingView, target ai.UnitView) ai.Order {
	return f.record(orderCall{Kind: 16, Subject: b.InternalID, Target: target.InternalID, Place: target.Location})
}
func (f *fakeView) BuildingAttackBuildingOrder(b ai.BuildingView, target ai.BuildingView) ai.Order {
	return f.record(orderCall{Kind: 17, Subject: b.InternalID, Target: target.InternalID, Place: target.Location})
}
func (f *fakeView) CancelFoundationOrder(f_view ai.FoundationView) ai.Order {
	return f.record(orderCall{Kind: 18, Item: f_view.BuildingID, Place: f_view.Location})
}
func (f *fakeView) HireMercenaryOrder(unit_id uint32, target ai.ZoneRef) ai.Order {
	return f.record(orderCall{Kind: 19, Item: unit_id, Place: target})
}
func (f *fakeView) CancelOrder(order_id uint64) ai.Order {
	return f.record(orderCall{Kind: 20, Target: order_id})
}
