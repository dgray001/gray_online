package ai

import "sort"

type gatherAction struct {
	category ResourceCategory
	eligible []OrderKind
	max      int
}

type balancedGatherAction struct {
	eligible []OrderKind
	weight   float64
}

type createAction struct {
	unit_id uint32
}

type researchAction struct {
	tech_id uint32
}

type buildAction struct {
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
	target   attackTarget
	max      int
	eligible []OrderKind
}

type attackSpaceAction struct {
	max      int
	eligible []OrderKind
}

type attackZoneAction struct {
	max      int
	eligible []OrderKind
}

type garrisonAction struct {
	building_id *uint32
	eligible    []OrderKind
	max         int
}

type ungarrisonAction struct {
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
	for _, u := range eligibleUnits(view, a.eligible) {
		if u.Kind != UnitEconomic {
			continue
		}
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
	pool := make([]UnitView, 0)
	for _, u := range eligibleUnits(view, a.eligible) {
		if u.Kind == UnitEconomic {
			pool = append(pool, u)
		}
	}
	if len(pool) == 0 {
		return nil
	}
	demand := gatherDemandWeights(view, internals, a.weight)
	counts := currentGatherCounts(view)
	orders := make([]Order, 0, len(pool))
	for _, u := range pool {
		category, target, ok := neediestGatherCategory(view, u.Location, demand, counts)
		if !ok {
			continue
		}
		if u.CurrentOrder != nil && u.CurrentOrder.TargetResource != nil {
			if u.CurrentOrder.TargetResource.Category == category {
				continue
			}
			counts[u.CurrentOrder.TargetResource.Category]--
		}
		orders = append(orders, view.GatherOrder(u, target, true))
		counts[category]++
	}
	return orders
}

func (a *createAction) ToOrders(view View, _ *Internals) []Order {
	orders := make([]Order, 0)
	current, limit := view.Population()
	for _, b := range view.IdleBuildings() {
		if current >= limit {
			break
		}
		for _, p := range b.Producibles {
			if p.Kind != ProducibleUnit || p.ID != a.unit_id || !canAfford(view, p.Cost) {
				continue
			}
			orders = append(orders, view.CreateUnitOrder(b, p.ID))
			current++
			break
		}
	}
	return orders
}

func (a *researchAction) ToOrders(view View, _ *Internals) []Order {
	orders := make([]Order, 0)
	for _, b := range view.IdleBuildings() {
		for _, p := range b.Producibles {
			if p.Kind != ProducibleTech || p.ID != a.tech_id || !canAfford(view, p.Cost) {
				continue
			}
			orders = append(orders, view.ResearchOrder(b, p.ID))
			break
		}
	}
	return orders
}

func (a *buildAction) ToOrders(view View, _ *Internals) []Order {
	orders := make([]Order, 0)
	if !canAfford(view, view.BuildCost(a.building_id)) {
		return orders
	}
	for _, u := range eligibleUnits(view, a.eligible) {
		if u.Kind != UnitEconomic {
			continue
		}
		if a.max > 0 && len(orders) >= a.max {
			break
		}
		if target, ok := view.NearestBuildSite(u.Location, a.building_id); ok {
			orders = append(orders, view.BuildOrder(u, a.building_id, target, true))
		}
	}
	return orders
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
	for _, u := range view.IdleUnits() {
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
	for _, u := range eligibleUnits(view, a.eligible) {
		if u.Kind != UnitMilitary {
			continue
		}
		if a.max > 0 && len(orders) >= a.max {
			break
		}
		if target, ok := a.pickTarget(view, u.Location, enemy_units); ok {
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
	for _, u := range eligibleUnits(view, a.eligible) {
		if u.Kind != UnitMilitary {
			continue
		}
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
	for _, u := range eligibleUnits(view, a.eligible) {
		if u.Kind != UnitMilitary {
			continue
		}
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
	buildings := view.Buildings()
	for _, u := range eligibleUnits(view, a.eligible) {
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
			if a.building_id != nil && b.BuildingID != *a.building_id {
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
	for _, u := range eligibleUnits(view, a.eligible) {
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
	weight     float64
	prioritize bool
	max        int
}

func (a *buildNextInQAction) ToOrders(view View, internals *Internals) []Order {
	q, ok := selectFromQueue(view, internals, a.weight, a.prioritize, QBuilding)
	if !ok {
		return nil
	}
	orders := make([]Order, 0)
	for _, u := range eligibleUnits(view, a.eligible) {
		if u.Kind != UnitEconomic {
			continue
		}
		if a.max > 0 && len(orders) >= a.max {
			break
		}
		if target, ok := view.NearestBuildSite(u.Location, *q.ID); ok {
			orders = append(orders, view.BuildOrder(u, *q.ID, target, true))
		}
	}
	return orders
}

func (a *createNextInQAction) ToOrders(view View, internals *Internals) []Order {
	q, ok := selectFromQueue(view, internals, a.weight, a.prioritize, QUnit)
	if !ok {
		return nil
	}
	orders := make([]Order, 0)
	current, limit := view.Population()
	for _, b := range view.IdleBuildings() {
		if current >= limit {
			break
		}
		for _, p := range b.Producibles {
			if p.Kind != ProducibleUnit || p.ID != *q.ID || !canAfford(view, p.Cost) {
				continue
			}
			orders = append(orders, view.CreateUnitOrder(b, p.ID))
			current++
			break
		}
	}
	return orders
}

func (a *researchNextInQAction) ToOrders(view View, internals *Internals) []Order {
	q, ok := selectFromQueue(view, internals, a.weight, a.prioritize, QTech)
	if !ok {
		return nil
	}
	orders := make([]Order, 0)
	for _, b := range view.IdleBuildings() {
		for _, p := range b.Producibles {
			if p.Kind != ProducibleTech || p.ID != *q.ID || !canAfford(view, p.Cost) {
				continue
			}
			orders = append(orders, view.ResearchOrder(b, p.ID))
			break
		}
	}
	return orders
}

func (a *produceNextInQAction) ToOrders(view View, internals *Internals) []Order {
	q, ok := selectFromQueue(view, internals, a.weight, a.prioritize, QBuilding, QUnit, QTech)
	if !ok {
		return nil
	}
	switch q.Type {
	case QBuilding:
		orders := make([]Order, 0)
		for _, u := range view.IdleUnits() {
			if u.Kind != UnitEconomic {
				continue
			}
			if a.max > 0 && len(orders) >= a.max {
				break
			}
			if target, ok := view.NearestBuildSite(u.Location, *q.ID); ok {
				orders = append(orders, view.BuildOrder(u, *q.ID, target, true))
			}
		}
		return orders
	case QUnit:
		orders := make([]Order, 0)
		current, limit := view.Population()
		for _, b := range view.IdleBuildings() {
			if current >= limit {
				break
			}
			for _, p := range b.Producibles {
				if p.Kind != ProducibleUnit || p.ID != *q.ID || !canAfford(view, p.Cost) {
					continue
				}
				orders = append(orders, view.CreateUnitOrder(b, p.ID))
				current++
				break
			}
		}
		return orders
	case QTech:
		orders := make([]Order, 0)
		for _, b := range view.IdleBuildings() {
			for _, p := range b.Producibles {
				if p.Kind != ProducibleTech || p.ID != *q.ID || !canAfford(view, p.Cost) {
					continue
				}
				orders = append(orders, view.ResearchOrder(b, p.ID))
				break
			}
		}
		return orders
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
	bucket   string
	eligible []OrderKind
	kind     *UnitKind
}

func (a *fillBucketAction) ToOrders(view View, internals *Internals) []Order {
	b := internals.bucket(a.bucket)
	need := b.Desired - len(b.Members)
	for _, u := range eligibleUnits(view, a.eligible) {
		if need <= 0 {
			break
		}
		if a.kind != nil && u.Kind != *a.kind {
			continue
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
