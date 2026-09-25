package ai

type fakeView struct {
	View
	units           []UnitView
	buildings       []BuildingView
	enemy_units     []UnitView
	enemy_buildings []BuildingView
	foundations     []FoundationView
	resources       map[ResourceCategory]float64
	population      int
	limit           int
	turn            int
	num_players     int
	enemies_found   int
	land            int
	score           int
	best_enemy      int
	techs           map[uint32]bool
	available       map[ResourceCategory]bool
	in_range        map[ZoneRef]bool
	build_site      ZoneRef
	costs           map[uint32]Cost
}

func (v *fakeView) Units() []UnitView                     { return v.units }
func (v *fakeView) Buildings() []BuildingView             { return v.buildings }
func (v *fakeView) IdleBuildings() []BuildingView         { return v.buildings }
func (v *fakeView) VisibleEnemyUnits() []UnitView         { return v.enemy_units }
func (v *fakeView) VisibleEnemyBuildings() []BuildingView { return v.enemy_buildings }
func (v *fakeView) Foundations() []FoundationView         { return v.foundations }
func (v *fakeView) Resource(c ResourceCategory) float64   { return v.resources[c] }
func (v *fakeView) Population() (int, int)                { return v.population, v.limit }
func (v *fakeView) TurnNumber() int                       { return v.turn }
func (v *fakeView) NumPlayers() int                       { return v.num_players }
func (v *fakeView) EnemiesFound() int                     { return v.enemies_found }
func (v *fakeView) OwnedSpaces() int                      { return v.land }
func (v *fakeView) Score() int                            { return v.score }
func (v *fakeView) BestEnemyScore() int                   { return v.best_enemy }
func (v *fakeView) TechResearched(id uint32) bool         { return v.techs[id] }
func (v *fakeView) RandomIntn(int) int                    { return 0 }
func (v *fakeView) BuildCost(id uint32) Cost              { return v.costs[id] }
func (v *fakeView) UnitCost(id uint32) Cost               { return v.costs[id] }
func (v *fakeView) TechCost(id uint32) Cost               { return v.costs[id] }

func (v *fakeView) InAttackRange(_ BuildingView, target ZoneRef) bool { return v.in_range[target] }

func (v *fakeView) NearestBuildSite(ZoneRef, uint32) (ZoneRef, bool) { return v.build_site, true }

func (v *fakeView) NearestResource(from ZoneRef, c ResourceCategory) (ResourceView, bool) {
	return ResourceView{Category: c, Location: ZoneRef{Zone: Coordinate{X: int(c) + 1}}}, v.available[c]
}

func (v *fakeView) IdleUnits() []UnitView { return v.EligibleUnits() }

func (v *fakeView) EligibleUnits(kinds ...OrderKind) []UnitView {
	allowed := make(map[OrderKind]bool)
	for _, k := range kinds {
		allowed[k] = true
	}
	out := make([]UnitView, 0)
	for _, u := range v.units {
		if u.CurrentOrder == nil || allowed[u.CurrentOrder.Kind] {
			out = append(out, u)
		}
	}
	return out
}

const (
	fakeRepair uint8 = iota + 100
	fakeDeleteUnit
	fakeDeleteBuilding
	fakeBuildingAttackUnit
	fakeBuildingAttackBuilding
	fakeCancel
	fakeCancelFoundation
	fakeBuild
	fakeRenew
	fakeGather
	fakeCreate
	fakeResearch
)

func unitOrder(t uint8, u UnitView, target int64) Order {
	return Order{Subjects: []uint64{u.InternalID}, OrderType: t, TargetID: target}
}

func (v *fakeView) RepairOrder(u UnitView, b BuildingView, _ bool) Order {
	return unitOrder(fakeRepair, u, int64(b.InternalID))
}
func (v *fakeView) RenewOrder(u UnitView, b BuildingView, _ bool) Order {
	return unitOrder(fakeRenew, u, int64(b.InternalID))
}
func (v *fakeView) DeleteUnitOrder(u UnitView) Order { return unitOrder(fakeDeleteUnit, u, 0) }
func (v *fakeView) GatherOrder(u UnitView, r ResourceView, _ bool) Order {
	return unitOrder(fakeGather, u, int64(r.Category))
}
func (v *fakeView) BuildOrder(u UnitView, building_id uint32, target ZoneRef, _ bool) Order {
	return unitOrder(fakeBuild, u, int64(building_id)*1000+int64(target.Space.X))
}
func (v *fakeView) DeleteBuildingOrder(b BuildingView) Order {
	return Order{Subjects: []uint64{b.InternalID}, OrderType: fakeDeleteBuilding}
}
func (v *fakeView) BuildingAttackUnitOrder(b BuildingView, target UnitView) Order {
	return Order{Subjects: []uint64{b.InternalID}, OrderType: fakeBuildingAttackUnit, TargetID: int64(target.InternalID)}
}
func (v *fakeView) BuildingAttackBuildingOrder(b BuildingView, target BuildingView) Order {
	return Order{Subjects: []uint64{b.InternalID}, OrderType: fakeBuildingAttackBuilding, TargetID: int64(target.InternalID)}
}
func (v *fakeView) CancelOrder(id uint64) Order {
	return Order{OrderType: fakeCancel, TargetID: int64(id)}
}
func (v *fakeView) CancelFoundationOrder(f FoundationView) Order {
	return Order{OrderType: fakeCancelFoundation, TargetID: int64(f.BuildingID)}
}
func (v *fakeView) CreateUnitOrder(b BuildingView, id uint32) Order {
	return Order{Subjects: []uint64{b.InternalID}, OrderType: fakeCreate, TargetID: int64(id)}
}
func (v *fakeView) ResearchOrder(b BuildingView, id uint32) Order {
	return Order{Subjects: []uint64{b.InternalID}, OrderType: fakeResearch, TargetID: int64(id)}
}
