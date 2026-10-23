package ai

import "cmp"

func (a *selectAction) ToOrders(view View, internals *Internals) []Order {
	cen := newCensus(view, internals)
	previous := internals.select_census
	internals.select_census = cen
	defer func() { internals.select_census = previous }()
	var targets []selItem
	if a.pick != nil {
		targets = a.pick.resolve(view, internals, cen)
	}
	if cen.invalid {
		return nil
	}
	if a.into != nil {
		a.into.store(a.pick.kind, targets, internals)
	}
	if len(a.verbs) == 0 {
		return nil
	}
	if a.hiring() {
		return a.hire(view, internals, targets)
	}
	if a.cancelling() {
		return a.cancelFoundations(view, internals, targets)
	}
	actors := a.actors.resolve(view, internals, cen)
	if cen.invalid {
		return nil
	}
	ctx := &issueContext{view: view, internals: internals}
	if !a.needsTarget() {
		orders := make([]Order, 0, len(actors))
		for _, actor := range actors {
			orders = append(orders, a.issueVerbs(ctx, actor, nil, true)...)
		}
		return orders
	}
	return a.assignAndIssue(ctx, actors, targets)
}

func (a *selectAction) issueVerbs(ctx *issueContext, actor loopItem, target *aim, first bool) []Order {
	orders := make([]Order, 0, len(a.verbs))
	clear := first && !a.queue
	for _, v := range a.verbs {
		if v.control() {
			orders = append(orders, ctx.control(v, actor)...)
		} else if v.production() {
			orders = append(orders, ctx.produce(v, actor)...)
		} else if order, ok := ctx.issue(v, actor, target, clear); ok {
			orders = append(orders, order)
			clear = false
		}
	}
	if a.prepend && !a.needsTarget() && first && len(orders) > 0 {
		if order, ok := ctx.requeue(actor); ok {
			orders = append(orders, order)
		}
	}
	return orders
}

type targetSlot struct {
	item     selItem
	aim      aim
	assigned float64
}

func (a *selectAction) slotsFor(targets []selItem) []targetSlot {
	slots := make([]targetSlot, 0, len(targets))
	for _, t := range targets {
		if aimed, ok := aimAt(t); ok {
			slots = append(slots, targetSlot{item: t, aim: aimed})
		}
	}
	return slots
}

func (a *selectAction) assignAndIssue(ctx *issueContext, actors []selItem, targets []selItem) []Order {
	slots := a.slotsFor(targets)
	if len(slots) == 0 || len(actors) == 0 {
		return nil
	}
	defer saveScoringContext(ctx.internals).restore(ctx.internals)
	rounds := 1
	previous := ctx.internals.eval_errors
	if a.assign != nil {
		rounds += max(0, a.assign.queue.int(ctx.view, ctx.internals))
	}
	if ctx.internals.eval_errors != previous {
		return nil
	}
	ordered := a.orderActors(ctx, actors)
	taken := make(map[uint64]map[int]bool, len(ordered))
	orders := make([]Order, 0, len(ordered))
	for round := 0; round < rounds; round++ {
		for _, actor := range ordered {
			key, _ := keyOf(actor)
			i, ok := a.pickSlot(ctx, actor, slots, taken[key.id], round)
			if !ok {
				continue
			}
			first := len(taken[key.id]) == 0
			previous := ctx.internals.eval_errors
			weight := a.weightOf(ctx, actor, slots[i])
			if ctx.internals.eval_errors != previous {
				continue
			}
			issued := a.issueVerbs(ctx, actor, &slots[i].aim, first)
			if len(issued) == 0 {
				continue
			}
			if taken[key.id] == nil {
				taken[key.id] = map[int]bool{}
			}
			taken[key.id][i] = true
			slots[i].assigned += weight
			orders = append(orders, issued...)
		}
	}
	if a.prepend {
		for _, actor := range ordered {
			key, _ := keyOf(actor)
			if len(taken[key.id]) > 0 {
				if order, ok := ctx.requeue(actor); ok {
					orders = append(orders, order)
				}
			}
		}
	}
	return orders
}

func (a *selectAction) orderActors(ctx *issueContext, actors []selItem) []selItem {
	if a.assign == nil {
		return actors
	}
	keys := make(map[uint64]float64, len(actors))
	for _, actor := range actors {
		restore := ctx.internals.bindItem("actor", actor)
		key, _ := keyOf(actor)
		previous := ctx.internals.eval_errors
		value := a.assign.unit_order.float(ctx.view, ctx.internals)
		if ctx.internals.eval_errors == previous {
			keys[key.id] = value
		}
		restore()
	}
	ordered := make([]selItem, 0, len(actors))
	for _, actor := range actors {
		key, _ := keyOf(actor)
		if _, valid := keys[key.id]; valid {
			ordered = append(ordered, actor)
		}
	}
	unitOrder := func(actor selItem) float64 { key, _ := keyOf(actor); return keys[key.id] }
	orderDrawingTies(ctx.internals, ordered, len(ordered),
		func(a, b selItem) int { return cmp.Compare(unitOrder(a), unitOrder(b)) },
		func(a, b selItem) bool { ka, _ := keyOf(a); kb, _ := keyOf(b); return ka.id < kb.id })
	return ordered
}

func (a *selectAction) weightOf(ctx *issueContext, actor selItem, slot targetSlot) float64 {
	if a.assign == nil {
		return 0
	}
	restore := a.bind(ctx, actor, slot, 0)
	defer restore()
	return max(0, a.assign.weight.float(ctx.view, ctx.internals))
}

func (a *selectAction) bind(ctx *issueContext, actor selItem, slot targetSlot, round int) func() {
	restore_actor := ctx.internals.bindItem("actor", actor)
	restore_target := ctx.internals.bindItem("target", slot.item)
	ctx.internals.target_assigned, ctx.internals.assign_round = slot.assigned, float64(round)
	return func() {
		restore_target()
		restore_actor()
	}
}
