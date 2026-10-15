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
		if !isMilitary(u) {
			continue
		}
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
