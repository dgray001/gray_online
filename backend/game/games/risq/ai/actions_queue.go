package ai

type addQAction struct {
	q_type QKind
	id     *uint32
	cost   Cost
	weight float64
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
	depth      int
	max        int
}

type createNextInQAction struct {
	weight     float64
	prioritize bool
	depth      int
}

type researchNextInQAction struct {
	weight     float64
	prioritize bool
	depth      int
}

type produceNextInQAction struct {
	filtered
	weight     float64
	prioritize bool
	depth      int
	max        int
}

func (a *buildNextInQAction) ToOrders(view View, internals *Internals) []Order {
	units := a.filter.apply(eligibleUnits(view, a.eligible), isEconomic)
	for _, q := range selectFromQueue(view, internals, a.weight, a.prioritize, a.depth, QBuilding) {
		if orders := buildWith(view, internals, units, *q.ID, a.max); len(orders) > 0 {
			return orders
		}
	}
	return nil
}

func (a *createNextInQAction) ToOrders(view View, internals *Internals) []Order {
	for _, q := range selectFromQueue(view, internals, a.weight, a.prioritize, a.depth, QUnit) {
		if orders := createUnits(view, internals, *q.ID, buildingFilter{}, 1); len(orders) > 0 {
			return orders
		}
	}
	return nil
}

func (a *researchNextInQAction) ToOrders(view View, internals *Internals) []Order {
	for _, q := range selectFromQueue(view, internals, a.weight, a.prioritize, a.depth, QTech) {
		if orders := researchTech(view, internals, *q.ID, buildingFilter{}, 1); len(orders) > 0 {
			return orders
		}
	}
	return nil
}

func (a *produceNextInQAction) ToOrders(view View, internals *Internals) []Order {
	units := a.filter.apply(view.IdleUnits(), isEconomic)
	for _, q := range selectFromQueue(view, internals, a.weight, a.prioritize, a.depth, QBuilding, QUnit, QTech) {
		var orders []Order
		switch q.Type {
		case QBuilding:
			orders = buildWith(view, internals, units, *q.ID, a.max)
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
