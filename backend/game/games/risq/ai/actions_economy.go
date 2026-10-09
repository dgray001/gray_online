package ai

type addQAction struct {
	q_type QKind
	id     *uint32
	cost   Cost
	weight amount
}

func (a *addQAction) ToOrders(view View, internals *Internals) []Order {
	q := Q{Type: a.q_type, Weight: a.weight.float(view, internals)}
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
	weight     amount
	prioritize bool
	depth      amount
	max        amount
}

type createNextInQAction struct {
	weight     amount
	prioritize bool
	depth      amount
}

type researchNextInQAction struct {
	weight     amount
	prioritize bool
	depth      amount
}

type produceNextInQAction struct {
	filtered
	weight     amount
	prioritize bool
	depth      amount
	max        amount
}

func (a *buildNextInQAction) ToOrders(view View, internals *Internals) []Order {
	limit := a.max.int(view, internals)
	units := a.filter.apply(eligibleUnits(view, a.eligible), isEconomic)
	for _, q := range selectFromQueue(view, internals, a.weight.float(view, internals), a.prioritize, a.depth.int(view, internals), QBuilding) {
		if orders := buildWith(view, internals, units, *q.ID, limit, nil, nil, true); len(orders) > 0 {
			return orders
		}
	}
	return nil
}

func (a *createNextInQAction) ToOrders(view View, internals *Internals) []Order {
	for _, q := range selectFromQueue(view, internals, a.weight.float(view, internals), a.prioritize, a.depth.int(view, internals), QUnit) {
		if orders := createUnits(view, internals, *q.ID, buildingFilter{}, 1); len(orders) > 0 {
			return orders
		}
	}
	return nil
}

func (a *researchNextInQAction) ToOrders(view View, internals *Internals) []Order {
	for _, q := range selectFromQueue(view, internals, a.weight.float(view, internals), a.prioritize, a.depth.int(view, internals), QTech) {
		if orders := researchTech(view, internals, *q.ID, buildingFilter{}, 1); len(orders) > 0 {
			return orders
		}
	}
	return nil
}

func (a *produceNextInQAction) ToOrders(view View, internals *Internals) []Order {
	limit := a.max.int(view, internals)
	units := a.filter.apply(view.IdleUnits(), isEconomic)
	for _, q := range selectFromQueue(view, internals, a.weight.float(view, internals), a.prioritize, a.depth.int(view, internals), QBuilding, QUnit, QTech) {
		var orders []Order
		switch q.Type {
		case QBuilding:
			orders = buildWith(view, internals, units, *q.ID, limit, nil, nil, true)
		case QUnit:
			orders = createUnits(view, internals, *q.ID, buildingFilter{}, 1)
		case QTech:
			orders = researchTech(view, internals, *q.ID, buildingFilter{}, 1)
		}
		if len(orders) > 0 {
			return orders
		}
	}
	return nil
}

// hireAction buys mercenaries with gold at the home village center once Mercenary Contracts is known.
// Mercenary price mirrors risq.mercenaryCost: (food + wood + stone + 1.5*gold) * 1.3, paid in gold.
type hireAction struct {
	unit_id uint32
	max     amount
	reserve amount
}

func (a *hireAction) ToOrders(view View, internals *Internals) []Order {
	limit, reserve := a.max.int(view, internals), a.reserve.float(view, internals)
	if !view.TechResearched(4) {
		return nil
	}
	home, ok := homeLocation(view)
	if !ok {
		return nil
	}
	cost := view.UnitCost(a.unit_id)
	price := (cost.Food + cost.Wood + cost.Stone + 1.5*cost.Gold) * 1.3
	orders := make([]Order, 0)
	for limit <= 0 || len(orders) < limit {
		current, limit := internals.population(view)
		if current >= limit || internals.available(view, ResourceGold) < price+reserve {
			break
		}
		internals.spend(Cost{Gold: price})
		internals.pending_population++
		orders = append(orders, view.HireMercenaryOrder(a.unit_id, home))
	}
	return orders
}
