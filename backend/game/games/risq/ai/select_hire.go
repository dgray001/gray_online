package ai

func availableMercenary(view View, unit_id uint32) (Producible, bool) {
	for _, m := range view.AvailableMercenaries() {
		if m.ID == unit_id {
			return m, true
		}
	}
	return Producible{}, false
}

func (a *selectAction) hire(view View, internals *Internals, targets []selItem) []Order {
	v := a.verbs[0]
	mercenary, ok := availableMercenary(view, v.unit_id)
	if !ok {
		return nil
	}
	ctx := &issueContext{view: view, internals: internals}
	orders := make([]Order, 0)
	for _, target := range targets {
		orders = append(orders, ctx.hireAt(v, mercenary, target)...)
	}
	return orders
}

func (c *issueContext) hireAt(v verb, mercenary Producible, target selItem) []Order {
	aimed, ok := aimAt(target)
	if !ok {
		return nil
	}
	item := selItem{loopItem: mercenaryItem{producible: mercenary}, view: c.view, internals: c.internals, role: "item", cache: map[string]float64{}}
	defer c.internals.bindItem("item", item)()
	defer c.internals.bindItem("target", target)()
	previous := c.internals.eval_errors
	count := max(0, v.count.int(c.view, c.internals))
	if c.internals.eval_errors != previous {
		return nil
	}
	orders := make([]Order, 0, count)
	for n := 0; n < count; n++ {
		if order, ok := c.issue(v, item, &aimed, false); ok {
			orders = append(orders, order)
		}
	}
	return orders
}
