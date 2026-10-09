package unit

import (
	"github.com/dgray001/gray_online/game/games/risq/ai"
)

type fakeView struct {
	nickname                     string
	units                        []ai.UnitView
	buildings                    []ai.BuildingView
	resources                    map[ai.ResourceCategory]float64
	popCurrent, popLimit, popMax int
	foundations                  []ai.FoundationView

	visibleEnemyUnits   []ai.UnitView
	knownEnemyBuildings []ai.BuildingView

	turnNumber, mapSize, numPlayers, enemiesFound, ownedSpaces int
	allSpaces                                                  []ai.SpaceInfo
	allZones                                                   []ai.ZoneInfo
	playerIDs                                                  []int
	playerID                                                   int
	playerScores                                               map[int]int
	score, bestEnemyScore                                      int

	techResearched    map[uint32]bool
	buildingAvailable map[uint32]bool
	buildCost         map[uint32]ai.Cost
	unitCost          map[uint32]ai.Cost
	techCost          map[uint32]ai.Cost

	knownResources []ai.ResourceView
}

func (f *fakeView) Nickname() string                                  { return f.nickname }
func (f *fakeView) Units() []ai.UnitView                              { return f.units }
func (f *fakeView) ScopedUnits() []ai.UnitView                        { return f.units }
func (f *fakeView) IdleUnits() []ai.UnitView                          { return f.units }
func (f *fakeView) EligibleUnits(kinds ...ai.OrderKind) []ai.UnitView { return f.units }
func (f *fakeView) Buildings() []ai.BuildingView                      { return f.buildings }
func (f *fakeView) IdleBuildings() []ai.BuildingView                  { return f.buildings }
func (f *fakeView) Resource(category ai.ResourceCategory) float64     { return f.resources[category] }
func (f *fakeView) Population() (current, limit int)                  { return f.popCurrent, f.popLimit }
func (f *fakeView) MaxPopulation() int                                { return f.popMax }
func (f *fakeView) Foundations() []ai.FoundationView                  { return f.foundations }

func (f *fakeView) VisibleEnemyUnits() []ai.UnitView         { return f.visibleEnemyUnits }
func (f *fakeView) VisibleEnemyBuildings() []ai.BuildingView { return f.knownEnemyBuildings }
func (f *fakeView) KnownEnemyBuildings() []ai.BuildingView   { return f.knownEnemyBuildings }

func (f *fakeView) NearestResource(from ai.ZoneRef, category ai.ResourceCategory, gatherer_id uint64) (ai.ResourceView, bool) {
	for _, r := range f.knownResources {
		if r.Category == category {
			return r, true
		}
	}
	return ai.ResourceView{}, false
}
func (f *fakeView) KnownResources(category ai.ResourceCategory) []ai.ResourceView {
	var res []ai.ResourceView
	for _, r := range f.knownResources {
		if r.Category == category {
			res = append(res, r)
		}
	}
	return res
}
func (f *fakeView) NearestBuildSite(from ai.ZoneRef, building_id uint32, include func(ai.Coordinate) bool) (ai.ZoneRef, bool) {
	return ai.ZoneRef{Space: ai.Coordinate{1, 1}}, true
}
func (f *fakeView) BuildSites(building_id uint32, include func(ai.Coordinate) bool) []ai.ZoneRef {
	return nil
}
func (f *fakeView) OwnBuildingsIn(space ai.Coordinate) int { return 0 }
func (f *fakeView) AssignedTo(building bool, internal_id uint64, exclude map[uint64]bool) int {
	return 0
}
func (f *fakeView) EnemyDistance(from ai.Coordinate) int       { return -1 }
func (f *fakeView) RegionProgress(space ai.Coordinate) float64 { return 0 }
func (f *fakeView) NearestUnexplored(from ai.ZoneRef) ([]ai.ZoneRef, bool) {
	return []ai.ZoneRef{{}}, true
}
func (f *fakeView) TurnNumber() int                                            { return f.turnNumber }
func (f *fakeView) MapSize() int                                               { return f.mapSize }
func (f *fakeView) NumPlayers() int                                            { return f.numPlayers }
func (f *fakeView) EnemiesFound() int                                          { return f.enemiesFound }
func (f *fakeView) OwnedSpaces() int                                           { return f.ownedSpaces }
func (f *fakeView) AllSpaces() []ai.SpaceInfo                                  { return f.allSpaces }
func (f *fakeView) AllZones() []ai.ZoneInfo                                    { return f.allZones }
func (f *fakeView) PlayerIDs() []int                                           { return f.playerIDs }
func (f *fakeView) PlayerID() int                                              { return f.playerID }
func (f *fakeView) PlayerScore(player_id int) int                              { return f.playerScores[player_id] }
func (f *fakeView) MatchingSpaces(condition ai.SpaceCondition) []ai.Coordinate { return nil }
func (f *fakeView) CountSpaces(condition ai.SpaceCondition, include func(ai.Coordinate) bool) int {
	return 0
}
func (f *fakeView) ClosestSpaces(reference ai.Coordinate, condition ai.SpaceCondition, exclude_reference bool) []ai.Coordinate {
	return nil
}
func (f *fakeView) SpaceDistance(a ai.Coordinate, b ai.Coordinate) int { return 0 }
func (f *fakeView) LocationDistance(a ai.ZoneRef, b ai.ZoneRef) int    { return 0 }
func (f *fakeView) Score() int                                         { return f.score }
func (f *fakeView) BestEnemyScore() int                                { return f.bestEnemyScore }
func (f *fakeView) TechResearched(tech_id uint32) bool                 { return f.techResearched[tech_id] }
func (f *fakeView) BuildingAvailable(building_id uint32) bool {
	return f.buildingAvailable[building_id]
}
func (f *fakeView) InAttackRange(b ai.BuildingView, target ai.ZoneRef) bool { return true }
func (f *fakeView) BuildCost(building_id uint32) ai.Cost                    { return f.buildCost[building_id] }
func (f *fakeView) UnitCost(unit_id uint32) ai.Cost                         { return f.unitCost[unit_id] }
func (f *fakeView) TechCost(tech_id uint32) ai.Cost                         { return f.techCost[tech_id] }

func (f *fakeView) MoveOrder(u ai.UnitView, target ai.ZoneRef, clear_previous bool) ai.Order {
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: 1}
}
func (f *fakeView) GatherOrder(u ai.UnitView, target ai.ResourceView, clear_previous bool) ai.Order {
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: 2}
}
func (f *fakeView) BuildOrder(u ai.UnitView, building_id uint32, target ai.ZoneRef, clear_previous bool) ai.Order {
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: 3}
}
func (f *fakeView) RepairOrder(u ai.UnitView, target ai.BuildingView, clear_previous bool) ai.Order {
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: 4}
}
func (f *fakeView) RenewOrder(u ai.UnitView, target ai.BuildingView, clear_previous bool) ai.Order {
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: 5}
}
func (f *fakeView) AttackUnitOrder(u ai.UnitView, target ai.UnitView, clear_previous bool) ai.Order {
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: 6}
}
func (f *fakeView) AttackBuildingOrder(u ai.UnitView, target ai.BuildingView, clear_previous bool) ai.Order {
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: 7}
}
func (f *fakeView) AttackSpaceOrder(u ai.UnitView, target ai.Coordinate, clear_previous bool) ai.Order {
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: 8}
}
func (f *fakeView) AttackZoneOrder(u ai.UnitView, target ai.ZoneRef, clear_previous bool) ai.Order {
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: 9}
}
func (f *fakeView) GarrisonOrder(u ai.UnitView, target ai.BuildingView, clear_previous bool) ai.Order {
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: 10}
}
func (f *fakeView) UngarrisonOrder(u ai.UnitView, clear_previous bool) ai.Order {
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: 11}
}
func (f *fakeView) DeleteUnitOrder(u ai.UnitView) ai.Order {
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: 12}
}
func (f *fakeView) CreateUnitOrder(b ai.BuildingView, unit_id uint32) ai.Order {
	return ai.Order{Subjects: []uint64{b.InternalID}, OrderType: 13}
}
func (f *fakeView) ResearchOrder(b ai.BuildingView, tech_id uint32) ai.Order {
	return ai.Order{Subjects: []uint64{b.InternalID}, OrderType: 14}
}
func (f *fakeView) DeleteBuildingOrder(b ai.BuildingView) ai.Order {
	return ai.Order{Subjects: []uint64{b.InternalID}, OrderType: 15}
}
func (f *fakeView) BuildingAttackUnitOrder(b ai.BuildingView, target ai.UnitView) ai.Order {
	return ai.Order{Subjects: []uint64{b.InternalID}, OrderType: 16}
}
func (f *fakeView) BuildingAttackBuildingOrder(b ai.BuildingView, target ai.BuildingView) ai.Order {
	return ai.Order{Subjects: []uint64{b.InternalID}, OrderType: 17}
}
func (f *fakeView) CancelFoundationOrder(f_view ai.FoundationView) ai.Order {
	return ai.Order{OrderType: 18}
}
func (f *fakeView) HireMercenaryOrder(unit_id uint32, target ai.ZoneRef) ai.Order {
	return ai.Order{OrderType: 19}
}
func (f *fakeView) CancelOrder(order_id uint64) ai.Order { return ai.Order{OrderType: 20} }
