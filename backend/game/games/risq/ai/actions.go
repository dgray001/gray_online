package ai

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
		if u.CurrentOrder != nil && u.CurrentOrder.GatherCategory != nil {
			if *u.CurrentOrder.GatherCategory == category {
				continue
			}
			counts[*u.CurrentOrder.GatherCategory]--
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
		target, ok := nearestZone(view, u.Location, candidates)
		if !ok {
			continue
		}
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
			orders = append(orders, view.AttackUnitOrder(u, target.InternalID, true))
		} else if target, ok := nearestBuilding(view, u.Location, enemy_buildings); ok {
			orders = append(orders, view.AttackBuildingOrder(u, target.InternalID, true))
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
			orders = append(orders, view.GarrisonOrder(u, best_b.InternalID, true))
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
