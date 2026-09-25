package ai

import (
	"slices"
	"sort"
)

type gatherAction struct {
	filtered
	category ResourceCategory
	eligible []OrderKind
	max      int
}

type balancedGatherAction struct {
	filtered
	eligible     []OrderKind
	weight       float64
	move_penalty float64
}

type createAction struct {
	unit_id uint32
}

type researchAction struct {
	tech_id uint32
}

type buildAction struct {
	filtered
	building_id uint32
	eligible    []OrderKind
	max         int
}

type exploreAnchor uint8

const (
	exploreAnchorSelf exploreAnchor = iota
	exploreAnchorHome
	exploreAnchorCenter
)

type exploreAction struct {
	filtered
	max    int
	anchor exploreAnchor
}

type attackTarget uint8

const (
	attackTargetMilitary attackTarget = iota
	attackTargetEconomic
	attackTargetAny
)

type attackAction struct {
	filtered
	target   attackTarget
	max      int
	eligible []OrderKind
}

type attackSpaceAction struct {
	filtered
	max      int
	eligible []OrderKind
}

type attackZoneAction struct {
	filtered
	max      int
	eligible []OrderKind
}

type garrisonAction struct {
	filtered
	buildingFiltered
	eligible []OrderKind
	max      int
}

type ungarrisonAction struct {
	filtered
	eligible []OrderKind
	max      int
}

type addQAction struct {
	q_type QKind
	id     *uint32
	cost   Cost
	weight float64
}

func (a *gatherAction) ToOrders(view View, _ *Internals) []Order {
	orders := make([]Order, 0)
	for _, u := range a.filter.apply(eligibleUnits(view, a.eligible), isEconomic) {
		if a.max > 0 && len(orders) >= a.max {
			break
		}
		if target, ok := view.NearestResource(u.Location, a.category); ok {
			orders = append(orders, view.GatherOrder(u, target, true))
		}
	}
	return orders
}

func (a *balancedGatherAction) ToOrders(view View, internals *Internals) []Order {
	idle, gathering := make([]UnitView, 0), make([]UnitView, 0)
	for _, u := range a.filter.apply(eligibleUnits(view, a.eligible), isEconomic) {
		if u.CurrentOrder != nil && u.CurrentOrder.TargetResource != nil {
			gathering = append(gathering, u)
		} else {
			idle = append(idle, u)
		}
	}
	counts := currentGatherCounts(view)
	targets := gatherTargets(gatherDemandWeights(view, internals, a.weight), counts, len(idle))
	orders := make([]Order, 0)
	for _, u := range idle {
		category, target, ok := neediestGatherCategory(view, u.Location, targets, counts)
		if ok {
			orders = append(orders, view.GatherOrder(u, target, true))
			counts[category]++
		}
	}
	deficit := func(c ResourceCategory) float64 { return targets[c] - float64(counts[c]) }
	for _, u := range gathering {
		from := u.CurrentOrder.TargetResource.Category
		category, target, ok := neediestGatherCategory(view, u.Location, targets, counts)
		if !ok || category == from || deficit(category)-deficit(from) <= 2+a.move_penalty {
			continue
		}
		orders = append(orders, view.GatherOrder(u, target, true))
		counts[from]--
		counts[category]++
	}
	return orders
}

func (a *createAction) ToOrders(view View, internals *Internals) []Order {
	return createUnits(view, internals, a.unit_id)
}

func (a *researchAction) ToOrders(view View, internals *Internals) []Order {
	return researchTech(view, internals, a.tech_id)
}

func (a *buildAction) ToOrders(view View, internals *Internals) []Order {
	return buildWith(view, internals, a.filter.apply(eligibleUnits(view, a.eligible), isEconomic), a.building_id, a.max)
}

func (a *exploreAction) ToOrders(view View, _ *Internals) []Order {
	orders := make([]Order, 0)
	var anchor ZoneRef
	fixed := a.anchor != exploreAnchorSelf
	if a.anchor == exploreAnchorHome {
		home, ok := homeLocation(view)
		if !ok {
			return orders
		}
		anchor = home
	}
	claimed := make(map[ZoneRef]bool)
	for _, u := range view.Units() {
		if u.CurrentOrder != nil && u.CurrentOrder.Kind == OrderKindMove && u.CurrentOrder.TargetZone != nil {
			claimed[*u.CurrentOrder.TargetZone] = true
		}
	}
	for _, u := range a.filter.apply(view.IdleUnits(), anyUnit) {
		if a.max > 0 && len(orders) >= a.max {
			break
		}
		from := u.Location
		if fixed {
			from = anchor
		}
		candidates, ok := view.NearestUnexplored(from)
		if !ok {
			continue
		}
		available := make([]ZoneRef, 0, len(candidates))
		for _, c := range candidates {
			if !claimed[c] {
				available = append(available, c)
			}
		}
		if len(available) == 0 {
			available = candidates
		}
		target, ok := nearestZone(view, u.Location, available)
		if !ok {
			continue
		}
		claimed[target] = true
		orders = append(orders, view.MoveOrder(u, target, true))
	}
	return orders
}

func (a *attackAction) ToOrders(view View, _ *Internals) []Order {
	orders := make([]Order, 0)
	enemy_units := view.VisibleEnemyUnits()
	enemy_buildings := view.VisibleEnemyBuildings()
	for _, u := range a.filter.apply(eligibleUnits(view, a.eligible), isMilitary) {
		if a.max > 0 && len(orders) >= a.max {
			break
		}
		if target, ok := pickUnitTarget(view, a.target, u.Location, enemy_units); ok {
			orders = append(orders, view.AttackUnitOrder(u, target, true))
		} else if target, ok := nearestBuilding(view, u.Location, enemy_buildings); ok {
			orders = append(orders, view.AttackBuildingOrder(u, target, true))
		}
	}
	return orders
}

func (a *attackSpaceAction) ToOrders(view View, _ *Internals) []Order {
	orders := make([]Order, 0)
	buildings := view.VisibleEnemyBuildings()
	for _, u := range a.filter.apply(eligibleUnits(view, a.eligible), isMilitary) {
		if a.max > 0 && len(orders) >= a.max {
			break
		}
		if target, ok := nearestEnemySpace(view, u.Location, buildings); ok {
			orders = append(orders, view.AttackSpaceOrder(u, target, true))
		}
	}
	return orders
}

func (a *attackZoneAction) ToOrders(view View, _ *Internals) []Order {
	orders := make([]Order, 0)
	units := view.VisibleEnemyUnits()
	buildings := view.VisibleEnemyBuildings()
	for _, u := range a.filter.apply(eligibleUnits(view, a.eligible), isMilitary) {
		if a.max > 0 && len(orders) >= a.max {
			break
		}
		if target, ok := nearestEnemyZone(view, u.Location, units, buildings); ok {
			orders = append(orders, view.AttackZoneOrder(u, target, true))
		}
	}
	return orders
}

func (a *garrisonAction) ToOrders(view View, _ *Internals) []Order {
	orders := make([]Order, 0)
	buildings := a.buildings.apply(view.Buildings())
	for _, u := range a.filter.apply(eligibleUnits(view, a.eligible), anyUnit) {
		if u.GarrisonedIn != nil {
			continue
		}
		if a.max > 0 && len(orders) >= a.max {
			break
		}
		var best_b *BuildingView
		best_dist := -1
		for _, b := range buildings {
			if b.UnderConstruction || b.GarrisonCount >= b.GarrisonCapacity {
				continue
			}
			d := locationDistance(u.Location, b.Location)
			if best_dist == -1 || d < best_dist {
				best_b, best_dist = &b, d
			}
		}
		if best_b != nil {
			orders = append(orders, view.GarrisonOrder(u, *best_b, true))
			best_b.GarrisonCount++ // Optimistic update for other units in this turn
		}
	}
	return orders
}

func (a *ungarrisonAction) ToOrders(view View, _ *Internals) []Order {
	orders := make([]Order, 0)
	for _, u := range a.filter.apply(eligibleUnits(view, a.eligible), anyUnit) {
		if u.GarrisonedIn == nil {
			continue
		}
		if a.max > 0 && len(orders) >= a.max {
			break
		}
		orders = append(orders, view.UngarrisonOrder(u, true))
	}
	return orders
}

func (a *addQAction) ToOrders(view View, internals *Internals) []Order {
	q := Q{Type: a.q_type, Weight: a.weight}
	switch a.q_type {
	case QResource:
		q.Cost = a.cost
	case QUnit:
		q.ID = a.id
		q.Cost = view.UnitCost(*a.id)
	case QBuilding:
		q.ID = a.id
		q.Cost = view.BuildCost(*a.id)
	case QTech:
		q.ID = a.id
		q.Cost = view.TechCost(*a.id)
	}
	internals.q = append(internals.q, q)
	return nil
}

type buildNextInQAction struct {
	filtered
	eligible   []OrderKind
	weight     float64
	prioritize bool
	max        int
}

type createNextInQAction struct {
	weight     float64
	prioritize bool
}

type researchNextInQAction struct {
	weight     float64
	prioritize bool
}

type produceNextInQAction struct {
	filtered
	weight     float64
	prioritize bool
	max        int
}

func (a *buildNextInQAction) ToOrders(view View, internals *Internals) []Order {
	q, ok := selectFromQueue(view, internals, a.weight, a.prioritize, QBuilding)
	if !ok {
		return nil
	}
	return buildWith(view, internals, a.filter.apply(eligibleUnits(view, a.eligible), isEconomic), *q.ID, a.max)
}

func (a *createNextInQAction) ToOrders(view View, internals *Internals) []Order {
	q, ok := selectFromQueue(view, internals, a.weight, a.prioritize, QUnit)
	if !ok {
		return nil
	}
	return createUnits(view, internals, *q.ID)
}

func (a *researchNextInQAction) ToOrders(view View, internals *Internals) []Order {
	q, ok := selectFromQueue(view, internals, a.weight, a.prioritize, QTech)
	if !ok {
		return nil
	}
	return researchTech(view, internals, *q.ID)
}

func (a *produceNextInQAction) ToOrders(view View, internals *Internals) []Order {
	q, ok := selectFromQueue(view, internals, a.weight, a.prioritize, QBuilding, QUnit, QTech)
	if !ok {
		return nil
	}
	switch q.Type {
	case QBuilding:
		return buildWith(view, internals, a.filter.apply(view.IdleUnits(), isEconomic), *q.ID, a.max)
	case QUnit:
		return createUnits(view, internals, *q.ID)
	case QTech:
		return researchTech(view, internals, *q.ID)
	}
	return nil
}

type unbucketedAction struct {
	inner Action
}

func (a *unbucketedAction) ToOrders(view View, internals *Internals) []Order {
	return a.inner.ToOrders(&unbucketedView{View: view, internals: internals}, internals)
}

type setBucketAction struct {
	bucket string
	size   int
	task   Action
}

func (a *setBucketAction) ToOrders(_ View, internals *Internals) []Order {
	b := internals.bucket(a.bucket)
	b.Desired = a.size
	b.Task = a.task
	return nil
}

type fillBucketAction struct {
	filtered
	bucket   string
	eligible []OrderKind
}

func (a *fillBucketAction) ToOrders(view View, internals *Internals) []Order {
	b := internals.bucket(a.bucket)
	need := b.Desired - len(b.Members)
	for _, u := range a.filter.apply(eligibleUnits(view, a.eligible), anyUnit) {
		if need <= 0 {
			break
		}
		if internals.isBucketed(u.InternalID) {
			continue
		}
		b.Members[u.InternalID] = true
		need--
	}
	return nil
}

type runBucketAction struct {
	bucket    string
	when_full bool
}

func (a *runBucketAction) ToOrders(view View, internals *Internals) []Order {
	b := internals.Buckets[a.bucket]
	if b == nil || b.Task == nil || len(b.Members) == 0 {
		return nil
	}
	if a.when_full && len(b.Members) < b.Desired {
		return nil
	}
	return b.Task.ToOrders(&bucketView{View: view, members: b.Members}, internals)
}

type emptyBucketAction struct {
	bucket string
}

func (a *emptyBucketAction) ToOrders(_ View, internals *Internals) []Order {
	if b := internals.Buckets[a.bucket]; b != nil {
		b.Members = make(map[uint64]bool)
	}
	return nil
}

type drainBucketAction struct {
	to       string
	from     []string
	from_any bool
	max      int
}

func (a *drainBucketAction) ToOrders(_ View, internals *Internals) []Order {
	to := internals.bucket(a.to)
	need := to.Desired - len(to.Members)
	if need <= 0 {
		return nil
	}
	source_names := a.from
	if a.from_any {
		source_names = make([]string, 0, len(internals.Buckets))
		for name := range internals.Buckets {
			if name != a.to {
				source_names = append(source_names, name)
			}
		}
		sort.Strings(source_names)
	}
	taken := 0
	for _, name := range source_names {
		from := internals.Buckets[name]
		if from == nil {
			continue
		}
		overflow := len(from.Members) - from.Desired
		ids := make([]uint64, 0, len(from.Members))
		for id := range from.Members {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		for _, id := range ids {
			if need <= 0 || overflow <= 0 || (a.max > 0 && taken >= a.max) {
				break
			}
			delete(from.Members, id)
			to.Members[id] = true
			need--
			overflow--
			taken++
		}
		if need <= 0 || (a.max > 0 && taken >= a.max) {
			break
		}
	}
	return nil
}

type buildFoundationsAction struct {
	filtered
	buildingFiltered
	eligible []OrderKind
	max      int
}

func (a *buildFoundationsAction) ToOrders(view View, _ *Internals) []Order {
	units := a.filter.apply(eligibleUnits(view, a.eligible), isEconomic)
	orders := make([]Order, 0)
	for _, f := range view.Foundations() {
		if f.Builders > 0 || !a.buildings.matches(f.BuildingID) {
			continue
		}
		if a.max > 0 && len(orders) >= a.max {
			break
		}
		if u, ok := nearestUnit(view, f.Location, units); ok {
			orders = append(orders, view.BuildOrder(u, f.BuildingID, f.Location, true))
			units = withoutUnit(units, u.InternalID)
		}
	}
	return orders
}

type cancelFoundationsAction struct {
	buildingFiltered
}

func (a *cancelFoundationsAction) ToOrders(view View, internals *Internals) []Order {
	orders := make([]Order, 0)
	for _, f := range view.Foundations() {
		if !f.Planned || f.Builders > 0 || !a.buildings.matches(f.BuildingID) {
			continue
		}
		cost := view.BuildCost(f.BuildingID)
		internals.spend(Cost{Food: -cost.Food, Wood: -cost.Wood, Stone: -cost.Stone, Gold: -cost.Gold})
		orders = append(orders, view.CancelFoundationOrder(f))
	}
	return orders
}

type renewAction struct {
	filtered
	buildingFiltered
	eligible []OrderKind
	max      int
}

func (a *renewAction) ToOrders(view View, internals *Internals) []Order {
	units := a.filter.apply(eligibleUnits(view, a.eligible), isEconomic)
	renewers := assignedBuildings(view, OrderKindRenew)
	orders := make([]Order, 0)
	for _, b := range a.buildings.apply(view.Buildings()) {
		if !b.Gatherable || b.UnderConstruction || b.ResourcesLeft > 0 || renewers[b.InternalID] || (!b.Renewing && !canAfford(view, internals, b.RenewCost)) {
			continue
		}
		if a.max > 0 && len(orders) >= a.max {
			break
		}
		if u, ok := nearestUnit(view, b.Location, units); ok {
			if !b.Renewing {
				internals.spend(b.RenewCost)
			}
			orders = append(orders, view.RenewOrder(u, b, true))
			units = withoutUnit(units, u.InternalID)
		}
	}
	return orders
}

type setUnitBehaviorAction struct {
	filtered
	behavior UnitBehavior
}

func (a *setUnitBehaviorAction) differs(u UnitView) bool {
	b := a.behavior
	return (b.Stance != nil && *b.Stance != u.Stance) ||
		(b.InterruptCurrent != nil && *b.InterruptCurrent != u.InterruptCurrent) ||
		(b.AttackBack != nil && *b.AttackBack != u.AttackBack) ||
		(b.TargetPriority != nil && !slices.Equal(b.TargetPriority, u.TargetPriority))
}

func (a *setUnitBehaviorAction) ToOrders(view View, internals *Internals) []Order {
	behavior := a.behavior
	for _, u := range a.filter.apply(view.Units(), isMilitary) {
		if a.differs(u) {
			behavior.Subjects = append(behavior.Subjects, u.InternalID)
		}
	}
	if len(behavior.Subjects) > 0 {
		internals.behaviors = append(internals.behaviors, behavior)
	}
	return nil
}

type repairAction struct {
	filtered
	buildingFiltered
	eligible []OrderKind
	max      int
}

func (a *repairAction) ToOrders(view View, _ *Internals) []Order {
	units := a.filter.apply(eligibleUnits(view, a.eligible), isEconomic)
	repairers := assignedBuildings(view, OrderKindRepair)
	orders := make([]Order, 0)
	for _, b := range a.buildings.apply(view.Buildings()) {
		if b.UnderConstruction || b.Health >= b.MaxHealth || repairers[b.InternalID] {
			continue
		}
		if a.max > 0 && len(orders) >= a.max {
			break
		}
		if u, ok := nearestUnit(view, b.Location, units); ok {
			orders = append(orders, view.RepairOrder(u, b, true))
			units = withoutUnit(units, u.InternalID)
		}
	}
	return orders
}

type deleteUnitAction struct {
	filtered
	eligible []OrderKind
	max      int
}

func (a *deleteUnitAction) ToOrders(view View, _ *Internals) []Order {
	orders := make([]Order, 0)
	for _, u := range a.filter.apply(eligibleUnits(view, a.eligible), anyUnit) {
		if a.max > 0 && len(orders) >= a.max {
			break
		}
		orders = append(orders, view.DeleteUnitOrder(u))
	}
	return orders
}

type deleteBuildingAction struct {
	buildingFiltered
	max int
}

func (a *deleteBuildingAction) ToOrders(view View, _ *Internals) []Order {
	orders := make([]Order, 0)
	for _, b := range a.buildings.apply(view.Buildings()) {
		if a.max > 0 && len(orders) >= a.max {
			break
		}
		orders = append(orders, view.DeleteBuildingOrder(b))
	}
	return orders
}

type buildingAttackAction struct {
	buildingFiltered
	target attackTarget
	max    int
}

func (a *buildingAttackAction) ToOrders(view View, _ *Internals) []Order {
	enemy_units, enemy_buildings := view.VisibleEnemyUnits(), view.VisibleEnemyBuildings()
	orders := make([]Order, 0)
	for _, b := range a.buildings.apply(view.Buildings()) {
		if !b.CanAttack || b.UnderConstruction || len(b.ActiveOrders) > 0 {
			continue
		}
		if a.max > 0 && len(orders) >= a.max {
			break
		}
		units, buildings := inAttackRange(view, b, enemy_units, enemy_buildings)
		if target, ok := pickUnitTarget(view, a.target, b.Location, units); ok {
			orders = append(orders, view.BuildingAttackUnitOrder(b, target))
		} else if target, ok := nearestBuilding(view, b.Location, buildings); ok {
			orders = append(orders, view.BuildingAttackBuildingOrder(b, target))
		}
	}
	return orders
}

type setBuildingBehaviorAction struct {
	buildingFiltered
	behavior BuildingBehavior
}

func (a *setBuildingBehaviorAction) differs(b BuildingView) bool {
	target := a.behavior
	return (target.AutoAttack != nil && *target.AutoAttack != b.AutoAttack) ||
		(target.InterruptCurrent != nil && *target.InterruptCurrent != b.InterruptCurrent) ||
		(target.TargetPriority != nil && !slices.Equal(target.TargetPriority, b.TargetPriority))
}

func (a *setBuildingBehaviorAction) ToOrders(view View, internals *Internals) []Order {
	behavior := a.behavior
	for _, b := range a.buildings.apply(view.Buildings()) {
		if b.CanAttack && a.differs(b) {
			behavior.Subjects = append(behavior.Subjects, b.InternalID)
		}
	}
	if len(behavior.Subjects) > 0 {
		internals.building_behaviors = append(internals.building_behaviors, behavior)
	}
	return nil
}

type unitStopAction struct {
	filtered
	orders map[OrderKind]bool
	max    int
}

func (a *unitStopAction) ToOrders(view View, _ *Internals) []Order {
	cancelled := make(map[uint64]bool)
	orders := make([]Order, 0)
	stopped := 0
	for _, u := range a.filter.apply(view.Units(), anyUnit) {
		if a.max > 0 && stopped >= a.max {
			break
		}
		before := len(orders)
		for _, o := range u.ActiveOrders {
			if (len(a.orders) == 0 || a.orders[o.Kind]) && !cancelled[o.ID] {
				cancelled[o.ID] = true
				orders = append(orders, view.CancelOrder(o.ID))
			}
		}
		if len(orders) > before {
			stopped++
		}
	}
	return orders
}

type buildingStopAction struct {
	buildingFiltered
	orders   map[BuildingOrderKind]bool
	item_ids map[uint32]bool
	max      int
}

func (a *buildingStopAction) ToOrders(view View, _ *Internals) []Order {
	orders := make([]Order, 0)
	stopped := 0
	for _, b := range a.buildings.apply(view.Buildings()) {
		if a.max > 0 && stopped >= a.max {
			break
		}
		before := len(orders)
		for _, o := range b.ActiveOrders {
			if (len(a.orders) == 0 || a.orders[o.Kind]) && (len(a.item_ids) == 0 || a.item_ids[o.ItemID]) {
				orders = append(orders, view.CancelOrder(o.ID))
			}
		}
		if len(orders) > before {
			stopped++
		}
	}
	return orders
}
