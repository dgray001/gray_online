package ai

import "slices"

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

func (a *attackAction) ToOrders(view View, internals *Internals) []Order {
	limit := a.max.int(view, internals)
	orders := make([]Order, 0)
	if a.picker != nil {
		units := a.filter.apply(eligibleUnits(view, a.eligible), isMilitary)
		if a.resume {
			units = slices.DeleteFunc(units, func(u UnitView) bool { return alreadyFighting(u.CurrentOrder) })
		}
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
			if b.UnderConstruction || b.GarrisonCount+internals.garrisons[b.InternalID] >= b.GarrisonCapacity {
				continue
			}
			d := view.LocationDistance(u.Location, b.Location)
			if best_dist == -1 || d < best_dist {
				best_b, best_dist = b, d
			}
		}
		if best_b != nil {
			orders = append(orders, view.GarrisonOrder(u, *best_b, true))
			internals.garrisons[best_b.InternalID]++
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
