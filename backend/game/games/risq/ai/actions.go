package ai

type gatherAction struct {
	category ResourceCategory
	eligible []OrderKind
}

type balancedGatherAction struct {
	eligible []OrderKind
}

type produceAction struct {
	unit_id uint32
}

type researchAction struct {
	tech_id uint32
}

type buildAction struct {
	building_id uint32
	eligible    []OrderKind
}

type exploreAction struct{}

type attackTarget uint8

const (
	attackTargetMilitary attackTarget = iota
	attackTargetEconomic
	attackTargetAny
)

type attackAction struct {
	target attackTarget
}

func (a *gatherAction) ToOrders(view View) []Order {
	orders := make([]Order, 0)
	for _, u := range eligibleUnits(view, a.eligible) {
		if u.Kind != UnitEconomic {
			continue
		}
		if target, ok := view.NearestResource(u.Location, a.category); ok {
			orders = append(orders, view.GatherOrder(u, target, true))
		}
	}
	return orders
}

func (a *balancedGatherAction) ToOrders(view View) []Order {
	pool := make([]UnitView, 0)
	for _, u := range eligibleUnits(view, a.eligible) {
		if u.Kind == UnitEconomic {
			pool = append(pool, u)
		}
	}
	if len(pool) == 0 {
		return nil
	}
	demand := gatherDemandWeights(view)
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

func (a *produceAction) ToOrders(view View) []Order {
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

func (a *researchAction) ToOrders(view View) []Order {
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

func (a *buildAction) ToOrders(view View) []Order {
	orders := make([]Order, 0)
	if !canAfford(view, view.BuildCost(a.building_id)) {
		return orders
	}
	for _, u := range eligibleUnits(view, a.eligible) {
		if u.Kind != UnitEconomic {
			continue
		}
		if target, ok := view.NearestBuildSite(u.Location, a.building_id); ok {
			orders = append(orders, view.BuildOrder(u, a.building_id, target, true))
		}
	}
	return orders
}

func (a *exploreAction) ToOrders(view View) []Order {
	orders := make([]Order, 0)
	for _, u := range view.IdleUnits() {
		if target, ok := view.NearestUnexplored(u.Location); ok {
			orders = append(orders, view.MoveOrder(u, target, true))
		}
	}
	return orders
}

func (a *attackAction) ToOrders(view View) []Order {
	orders := make([]Order, 0)
	enemy_units := view.VisibleEnemyUnits()
	enemy_buildings := view.VisibleEnemyBuildings()
	for _, u := range view.IdleUnits() {
		if u.Kind != UnitMilitary {
			continue
		}
		if target, ok := a.pickTarget(view, u.Location, enemy_units); ok {
			orders = append(orders, view.AttackUnitOrder(u, target.InternalID, true))
		} else if target, ok := nearestBuilding(view, u.Location, enemy_buildings); ok {
			orders = append(orders, view.AttackBuildingOrder(u, target.InternalID, true))
		}
	}
	return orders
}
