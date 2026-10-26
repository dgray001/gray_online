package unit

import (
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"slices"
	"testing"
)

func actionsModel(t *testing.T, actions string) ai.Model {
	t.Helper()
	return requireModel(t, `{"rules":[{"when":{"always":null},"then":[`+actions+`]}]}`)
}

func TestFixtureRejectsUnexpectedQuery(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("unconfigured query did not fail")
		}
	}()
	checkedView().KnownEnemyBuildings()
}

func TestFixtureQueriesReturnFreshSlices(t *testing.T) {
	view := checkedView("Buildings")
	units, buildings := view.Units(), view.Buildings()
	units[0].InternalID, buildings[0].InternalID = 0, 0
	if view.Units()[0].InternalID != 91 || view.Buildings()[0].InternalID != 800 {
		t.Fatal("query mutated fixture storage")
	}
}

func (f *fakeView) cancelLiveOrder(id uint64) {
	for index := range f.units {
		unit := &f.units[index]
		unit.ActiveOrders = slices.DeleteFunc(slices.Clone(unit.ActiveOrders), func(order ai.ActiveUnitOrder) bool { return order.ID == id })
		if len(unit.ActiveOrders) == 0 && f.assignedUnits[unit.InternalID] == nil {
			unit.CurrentOrder = nil
		}
		if len(unit.ActiveOrders) > 0 && f.queuedJobs != nil && f.assignedUnits[unit.InternalID] == nil {
			unit.CurrentOrder = f.queuedJobs[unit.ActiveOrders[0].ID]
		}
	}
	for index := range f.buildings {
		building := &f.buildings[index]
		building.ActiveOrders = slices.DeleteFunc(slices.Clone(building.ActiveOrders), func(order ai.ActiveBuildingOrder) bool { return order.ID == id })
		building.Idle = !building.UnderConstruction && len(building.ActiveOrders) == 0 && building.PlannedProduction == 0 && len(building.Producibles) > 0
	}
}

func checkedView(queries ...string) *fakeView {
	f := &fakeView{stateful: true, allowed: map[string]bool{}, popCurrent: 1, popLimit: 4, popMax: 10, turnNumber: 7, mapSize: 4, numPlayers: 3}
	for _, name := range append([]string{"Units", "SpaceDistance", "LocationDistance", "Resource", "Population"}, queries...) {
		f.allowed[name] = true
	}
	f.units = []ai.UnitView{{InternalID: 91, UnitID: 2, Kind: ai.UnitMilitary, Type: ai.UnitTypeInfantry, Builds: []ai.Producible{{Kind: ai.ProducibleBuilding, ID: 2}}}}
	f.buildings = []ai.BuildingView{{InternalID: 800, BuildingID: 1, Location: ai.ZoneRef{Zone: ai.Coordinate{X: 1}}, Idle: true, Health: 5, MaxHealth: 10, Gatherable: true, GarrisonCapacity: 1, CanAttack: true, Producibles: []ai.Producible{{Kind: ai.ProducibleUnit, ID: 2, Cost: ai.Cost{Food: 3}}, {Kind: ai.ProducibleTech, ID: 3, Cost: ai.Cost{Gold: 7}}}}}
	f.visibleEnemyUnits = []ai.UnitView{{InternalID: 201, UnitID: 3, Kind: ai.UnitMilitary, Type: ai.UnitTypeInfantry, Location: ai.ZoneRef{Space: ai.Coordinate{X: 4}}, Health: 8, MaxHealth: 10}}
	f.knownEnemyBuildings = []ai.BuildingView{{InternalID: 301, BuildingID: 2, Location: ai.ZoneRef{Space: ai.Coordinate{X: 5}}, Health: 6, MaxHealth: 12}}
	f.knownResources = []ai.ResourceView{{InternalID: 401, Category: ai.ResourceWood, Location: ai.ZoneRef{Space: ai.Coordinate{X: 3}}, AmountLeft: 20}}
	f.resources = map[ai.ResourceCategory]float64{ai.ResourceFood: 20, ai.ResourceWood: 30, ai.ResourceStone: 40, ai.ResourceGold: 50}
	f.unitCost, f.buildCost, f.techCost = map[uint32]ai.Cost{2: {Food: 3}}, map[uint32]ai.Cost{2: {Wood: 5}}, map[uint32]ai.Cost{3: {Gold: 7}}
	f.techResearched, f.buildingAvailable = map[uint32]bool{4: true}, map[uint32]bool{2: true}
	f.spaceDistances = map[[2]ai.Coordinate]int{{{}, {}}: 0, {{}, {X: 2}}: 2, {{}, {X: 3}}: 3, {{}, {X: 4}}: 4, {{}, {X: 5}}: 5}
	f.locationDistances = map[[2]ai.ZoneRef]int{{{}, {}}: 0, {{}, {Zone: ai.Coordinate{X: 1}}}: 1, {{}, {Space: ai.Coordinate{X: 2}}}: 4, {{}, {Space: ai.Coordinate{X: 3}}}: 6, {{}, {Space: ai.Coordinate{X: 4}}}: 8, {{}, {Space: ai.Coordinate{X: 5}}}: 10}
	return f
}
