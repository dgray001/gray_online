package ai

// hireAction buys mercenaries with gold at the home village center once Mercenary Contracts is known.
// Mercenary price mirrors risq.mercenaryCost: (food + wood + stone + 1.5*gold) * 1.3, paid in gold.
type hireAction struct {
	unit_id uint32
	max     int
	reserve float64
}

func (a *hireAction) ToOrders(view View, internals *Internals) []Order {
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
	for a.max <= 0 || len(orders) < a.max {
		current, limit := internals.population(view)
		if current >= limit || internals.available(view, ResourceGold) < price+a.reserve {
			break
		}
		internals.spend(Cost{Gold: price})
		internals.pending_population++
		orders = append(orders, view.HireMercenaryOrder(a.unit_id, home))
	}
	return orders
}
