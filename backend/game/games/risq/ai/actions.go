package ai

import (
	"slices"
)

type gatherAction struct {
	filtered
	category ResourceCategory
	eligible []OrderKind
	max      amount
}

type balancedGatherAction struct {
	filtered
	eligible     []OrderKind
	weight       amount
	move_penalty amount
}

type createAction struct {
	buildingFiltered
	unit_id uint32
	queue   amount
}

type researchAction struct {
	buildingFiltered
	tech_id uint32
	queue   amount
}

type buildAction struct {
	filtered
	building_id              uint32
	eligible                 []OrderKind
	max                      amount
	site                     *spaceQuery
	picker                   *sitePicker
	reuse_unbuilt_foundation bool
}

type exploreAnchor uint8

const (
	exploreAnchorSelf exploreAnchor = iota
	exploreAnchorHome
	exploreAnchorCenter
)

type exploreAction struct {
	filtered
	max          amount
	min_distance amount
	anchor       exploreAnchor
}

type attackTarget uint8

const (
	attackTargetMilitary attackTarget = iota
	attackTargetEconomic
	attackTargetAny
)

type attackAction struct {
	filtered
	picker   *targetPicker
	target   attackTarget
	max      amount
	eligible []OrderKind
	// queue each attacker's current job behind the attack, so it goes back to work once the fight is over
	resume bool
}

type attackSpaceAction struct {
	filtered
	max      amount
	eligible []OrderKind
}

type attackZoneAction struct {
	filtered
	max      amount
	eligible []OrderKind
}

type garrisonAction struct {
	filtered
	buildingFiltered
	eligible []OrderKind
	max      amount
}

type ungarrisonAction struct {
	filtered
	eligible []OrderKind
	max      amount
}

func (a *gatherAction) ToOrders(view View, internals *Internals) []Order {
	limit := a.max.int(view, internals)
	orders := make([]Order, 0)
	for _, u := range a.filter.apply(eligibleUnits(view, a.eligible), isEconomic) {
		if limit > 0 && len(orders) >= limit {
			break
		}
		if target, ok := view.NearestResource(u.Location, a.category, u.InternalID); ok {
			orders = append(orders, view.GatherOrder(u, target, true))
		}
	}
	return orders
}

func (a *balancedGatherAction) ToOrders(view View, internals *Internals) []Order {
	move_penalty := a.move_penalty.float(view, internals)
	idle, gathering := make([]UnitView, 0), make([]UnitView, 0)
	for _, u := range a.filter.apply(eligibleUnits(view, a.eligible), isEconomic) {
		if u.CurrentOrder != nil && u.CurrentOrder.TargetResource != nil {
			gathering = append(gathering, u)
		} else {
			idle = append(idle, u)
		}
	}
	counts := currentGatherCounts(view)
	targets := gatherTargets(gatherDemandWeights(view, internals, a.weight.float(view, internals)), counts, len(idle))
	orders := make([]Order, 0)
	for _, u := range idle {
		category, target, ok := neediestGatherCategory(view, u, targets, counts)
		if ok {
			orders = append(orders, view.GatherOrder(u, target, true))
			counts[category]++
		}
	}
	deficit := func(c ResourceCategory) float64 { return targets[c] - float64(counts[c]) }
	for _, u := range gathering {
		from := u.CurrentOrder.TargetResource.Category
		category, target, ok := neediestGatherCategory(view, u, targets, counts)
		if !ok || category == from || deficit(category)-deficit(from) <= 2+move_penalty {
			continue
		}
		orders = append(orders, view.GatherOrder(u, target, true))
		counts[from]--
		counts[category]++
	}
	return orders
}

func (a *createAction) ToOrders(view View, internals *Internals) []Order {
	return createUnits(view, internals, a.unit_id, a.buildings, a.queue.int(view, internals))
}

func (a *researchAction) ToOrders(view View, internals *Internals) []Order {
	return researchTech(view, internals, a.tech_id, a.buildings, a.queue.int(view, internals))
}

func (a *buildAction) ToOrders(view View, internals *Internals) []Order {
	limit := a.max.int(view, internals)
	units := a.filter.apply(eligibleUnits(view, a.eligible), isEconomic)
	return buildWith(view, internals, units, a.building_id, limit, a.site, a.picker, a.reuse_unbuilt_foundation)
}

func (a *exploreAction) ToOrders(view View, internals *Internals) []Order {
	limit := a.max.int(view, internals)
	min_distance := max(0, a.min_distance.int(view, internals))
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
		if limit > 0 && len(orders) >= limit {
			break
		}
		from := u.Location
		if fixed {
			from = anchor
		}
		candidates, ok := nearestUnexploredOutside(view, from, min_distance)
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
		target, ok := randomNearestSpace(view, internals, u.Location.Space, available)
		if !ok {
			continue
		}
		claimed[target] = true
		orders = append(orders, view.MoveOrder(u, target, true))
	}
	return orders
}

func (a *attackAction) ToOrders(view View, internals *Internals) []Order {
	limit := a.max.int(view, internals)
	orders := make([]Order, 0)
	if a.picker != nil {
		units := a.filter.apply(eligibleUnits(view, a.eligible), isMilitary)
		if limit > 0 && len(units) > limit {
			units = units[:limit]
		}
		orders := a.picker.pick(view, internals, units, func(u UnitView, c candidate, clear bool) Order {
			return attackCandidateOrder(view, u, c, a.picker.order, clear)
		})
		if a.resume {
			orders = withResumedJobs(view, units, orders)
		}
		return orders
	}
	enemy_units := view.VisibleEnemyUnits()
	enemy_buildings := view.VisibleEnemyBuildings()
	for _, u := range a.filter.apply(eligibleUnits(view, a.eligible), isMilitary) {
		if limit > 0 && len(orders) >= limit {
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

func (a *attackSpaceAction) ToOrders(view View, internals *Internals) []Order {
	limit := a.max.int(view, internals)
	orders := make([]Order, 0)
	buildings := view.VisibleEnemyBuildings()
	for _, u := range a.filter.apply(eligibleUnits(view, a.eligible), isMilitary) {
		if limit > 0 && len(orders) >= limit {
			break
		}
		if target, ok := nearestEnemySpace(view, u.Location, buildings); ok {
			orders = append(orders, view.AttackSpaceOrder(u, target, true))
		}
	}
	return orders
}

func (a *attackZoneAction) ToOrders(view View, internals *Internals) []Order {
	limit := a.max.int(view, internals)
	orders := make([]Order, 0)
	units := view.VisibleEnemyUnits()
	buildings := view.VisibleEnemyBuildings()
	for _, u := range a.filter.apply(eligibleUnits(view, a.eligible), isMilitary) {
		if limit > 0 && len(orders) >= limit {
			break
		}
		if target, ok := nearestEnemyZone(view, u.Location, units, buildings); ok {
			orders = append(orders, view.AttackZoneOrder(u, target, true))
		}
	}
	return orders
}

func (a *garrisonAction) ToOrders(view View, internals *Internals) []Order {
	limit := a.max.int(view, internals)
	orders := make([]Order, 0)
	buildings := a.buildings.apply(view.Buildings())
	for _, u := range a.filter.apply(eligibleUnits(view, a.eligible), anyUnit) {
		if u.GarrisonedIn != nil {
			continue
		}
		if limit > 0 && len(orders) >= limit {
			break
		}
		var best_b *BuildingView
		best_dist := -1
		for i := range buildings {
			b := &buildings[i]
			if b.UnderConstruction || b.GarrisonCount >= b.GarrisonCapacity {
				continue
			}
			d := view.LocationDistance(u.Location, b.Location)
			if best_dist == -1 || d < best_dist {
				best_b, best_dist = b, d
			}
		}
		if best_b != nil {
			orders = append(orders, view.GarrisonOrder(u, *best_b, true))
			best_b.GarrisonCount++ // Optimistic update for other units in this turn
		}
	}
	return orders
}

func (a *ungarrisonAction) ToOrders(view View, internals *Internals) []Order {
	limit := a.max.int(view, internals)
	orders := make([]Order, 0)
	for _, u := range a.filter.apply(eligibleUnits(view, a.eligible), anyUnit) {
		if u.GarrisonedIn == nil {
			continue
		}
		if limit > 0 && len(orders) >= limit {
			break
		}
		orders = append(orders, view.UngarrisonOrder(u, true))
	}
	return orders
}

type buildFoundationsAction struct {
	filtered
	buildingFiltered
	eligible []OrderKind
	max      amount
}

func (a *buildFoundationsAction) ToOrders(view View, internals *Internals) []Order {
	limit := a.max.int(view, internals)
	units := a.filter.apply(eligibleUnits(view, a.eligible), isEconomic)
	orders := make([]Order, 0)
	for _, f := range view.Foundations() {
		if f.Builders > 0 || !a.buildings.matches(f.BuildingID) {
			continue
		}
		if limit > 0 && len(orders) >= limit {
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
	max      amount
}

func (a *renewAction) ToOrders(view View, internals *Internals) []Order {
	limit := a.max.int(view, internals)
	units := a.filter.apply(eligibleUnits(view, a.eligible), isEconomic)
	renewers := assignedBuildings(view, OrderKindRenew)
	orders := make([]Order, 0)
	for _, b := range a.buildings.apply(view.Buildings()) {
		if !b.Gatherable || b.UnderConstruction || b.ResourcesLeft > 0 || renewers[b.InternalID] || (!b.Renewing && !canAfford(view, internals, b.RenewCost)) {
			continue
		}
		if limit > 0 && len(orders) >= limit {
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
	for _, u := range a.filter.apply(view.ScopedUnits(), isMilitary) {
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
	max      amount
}

func (a *repairAction) ToOrders(view View, internals *Internals) []Order {
	limit := a.max.int(view, internals)
	units := a.filter.apply(eligibleUnits(view, a.eligible), isEconomic)
	repairers := assignedBuildings(view, OrderKindRepair)
	orders := make([]Order, 0)
	for _, b := range a.buildings.apply(view.Buildings()) {
		if b.UnderConstruction || b.Health >= b.MaxHealth || repairers[b.InternalID] {
			continue
		}
		if limit > 0 && len(orders) >= limit {
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
	max      amount
}

func (a *deleteUnitAction) ToOrders(view View, internals *Internals) []Order {
	limit := a.max.int(view, internals)
	orders := make([]Order, 0)
	for _, u := range a.filter.apply(eligibleUnits(view, a.eligible), anyUnit) {
		if limit > 0 && len(orders) >= limit {
			break
		}
		orders = append(orders, view.DeleteUnitOrder(u))
	}
	return orders
}

type deleteBuildingAction struct {
	buildingFiltered
	max amount
}

func (a *deleteBuildingAction) ToOrders(view View, internals *Internals) []Order {
	limit := a.max.int(view, internals)
	orders := make([]Order, 0)
	for _, b := range a.buildings.apply(view.Buildings()) {
		if limit > 0 && len(orders) >= limit {
			break
		}
		orders = append(orders, view.DeleteBuildingOrder(b))
	}
	return orders
}

type buildingAttackAction struct {
	buildingFiltered
	target attackTarget
	max    amount
}

func (a *buildingAttackAction) ToOrders(view View, internals *Internals) []Order {
	limit := a.max.int(view, internals)
	enemy_units, enemy_buildings := view.VisibleEnemyUnits(), view.VisibleEnemyBuildings()
	orders := make([]Order, 0)
	for _, b := range a.buildings.apply(view.Buildings()) {
		if !b.CanAttack || b.UnderConstruction || len(b.ActiveOrders) > 0 {
			continue
		}
		if limit > 0 && len(orders) >= limit {
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
	max    amount
}

func (a *unitStopAction) ToOrders(view View, internals *Internals) []Order {
	limit := a.max.int(view, internals)
	cancelled := make(map[uint64]bool)
	orders := make([]Order, 0)
	stopped := 0
	for _, u := range a.filter.apply(view.ScopedUnits(), anyUnit) {
		if limit > 0 && stopped >= limit {
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
	max      amount
}

func (a *buildingStopAction) ToOrders(view View, internals *Internals) []Order {
	limit := a.max.int(view, internals)
	orders := make([]Order, 0)
	stopped := 0
	for _, b := range a.buildings.apply(view.Buildings()) {
		if limit > 0 && stopped >= limit {
			break
		}
		before := len(orders)
		for _, o := range b.ActiveOrders {
			if (len(a.orders) == 0 || a.orders[o.Kind]) && (len(a.item_ids) == 0 || a.item_ids[o.ItemID]) {
				var cost Cost
				switch o.Kind {
				case BuildingOrderCreate:
					cost = view.UnitCost(o.ItemID)
				case BuildingOrderResearch:
					cost = view.TechCost(o.ItemID)
				}
				internals.spend(Cost{Food: -cost.Food, Wood: -cost.Wood, Stone: -cost.Stone, Gold: -cost.Gold})
				orders = append(orders, view.CancelOrder(o.ID))
			}
		}
		if len(orders) > before {
			stopped++
		}
	}
	return orders
}

// After each attack order, the attacker's job from before it (gather, repair or renew) queued behind it
func withResumedJobs(view View, units []UnitView, orders []Order) []Order {
	before := make(map[uint64]*CurrentOrder, len(units))
	for _, u := range units {
		before[u.InternalID] = u.CurrentOrder
	}
	out := make([]Order, 0, 2*len(orders))
	for _, o := range orders {
		if len(o.Subjects) != 1 {
			out = append(out, o)
			continue
		}
		for _, u := range units {
			if u.InternalID != o.Subjects[0] {
				continue
			}
			// already fighting: leave it be, so the job queued behind its attack stays
			if alreadyFighting(before[u.InternalID]) {
				break
			}
			out = append(out, o)
			if job, ok := resumeOrder(view, u, before[u.InternalID]); ok {
				out = append(out, job)
			}
			break
		}
	}
	return out
}

func alreadyFighting(current *CurrentOrder) bool {
	if current == nil {
		return false
	}
	switch current.Kind {
	case OrderKindAttackSpace, OrderKindAttackUnit, OrderKindAttackZone, OrderKindAttackBuilding:
		return true
	}
	return false
}

func resumeOrder(view View, u UnitView, job *CurrentOrder) (Order, bool) {
	if job == nil {
		return Order{}, false
	}
	switch {
	case job.Kind == OrderKindGather && job.TargetResource != nil:
		return view.GatherOrder(u, *job.TargetResource, false), true
	case job.Kind == OrderKindRepair && job.TargetBuilding != nil:
		return view.RepairOrder(u, *job.TargetBuilding, false), true
	case job.Kind == OrderKindRenew && job.TargetBuilding != nil:
		return view.RenewOrder(u, *job.TargetBuilding, false), true
	case job.Kind == OrderKindBuild && job.TargetZone != nil:
		if building_id, ok := buildingAt(view, *job.TargetZone); ok {
			return view.BuildOrder(u, building_id, *job.TargetZone, false), true
		}
	}
	return Order{}, false
}

// What is being built at a zone: a foundation, or one of our buildings still under construction
func buildingAt(view View, zone ZoneRef) (uint32, bool) {
	for _, f := range view.Foundations() {
		if f.Location == zone {
			return f.BuildingID, true
		}
	}
	for _, b := range view.Buildings() {
		if b.Location == zone && b.UnderConstruction {
			return b.BuildingID, true
		}
	}
	return 0, false
}
