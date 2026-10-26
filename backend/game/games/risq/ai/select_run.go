package ai

import (
	"maps"
	"math"
	"slices"
)

func (a *selectAction) ToOrders(view View, internals *Internals) []Order {
	if a.carry != "" {
		return a.carryRound(view, internals)
	}
	cen := newCensus(view, internals)
	previous := internals.select_census
	internals.select_census = cen
	defer func() { internals.select_census = previous }()
	internals.target_queue_index = nil
	var job *assignJob
	if a.assign != "" {
		job = newAssignJob(a, cen)
		internals.storeAssignment(a.assign, job)
		defer func() {
			if cen.invalid {
				delete(internals.assignments, a.assign)
			}
		}()
	}
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
	if job != nil {
		return a.startAssignment(ctx, job, actors, targets)
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
	item       selItem
	aim        aim
	capacity   *float64
	remaining  *float64
	assigned   float64
	assign_key assignKey
}

func preEvalLets(item selItem, internals *Internals) bool {
	if item.lets == nil {
		return true
	}
	for _, name := range slices.Sorted(maps.Keys(item.lets.specs)) {
		spec := item.lets.specs[name]
		if spec.impure {
			continue
		}
		previous := internals.eval_errors
		item.letValue(name, spec)
		if internals.eval_errors != previous || item.census.invalid {
			return false
		}
	}
	return true
}

func (a *selectAction) slotsFor(ctx *issueContext, targets []selItem) []*targetSlot {
	slots := make([]*targetSlot, 0, len(targets))
	for _, t := range targets {
		if aimed, ok := aimAt(t); ok {
			var capPtr, remPtr *float64
			initial := 0.0
			if t.initial != nil {
				initial = *t.initial
			}
			if t.capacity != nil {
				capPtr = t.capacity
				rem := *t.capacity - initial
				remPtr = &rem
			}
			slot := &targetSlot{item: t, aim: aimed, capacity: capPtr, remaining: remPtr, assigned: initial, assign_key: assignKeyOf(t)}
			slot.item.slot = slot
			if !preEvalLets(slot.item, ctx.internals) {
				if slot.item.census.invalid {
					return nil
				}
				continue
			}
			slots = append(slots, slot)
		}
	}
	return slots
}

func (a *selectAction) assignAndIssue(ctx *issueContext, actors []selItem, targets []selItem) []Order {
	slots := a.slotsFor(ctx, targets)
	if len(slots) == 0 || len(actors) == 0 {
		return nil
	}
	readyActors := make([]selItem, 0, len(actors))
	for _, actor := range actors {
		if !preEvalLets(actor, ctx.internals) {
			if actor.census.invalid {
				return nil
			}
			continue
		}
		readyActors = append(readyActors, actor)
	}
	actors = readyActors
	if len(actors) == 0 {
		return nil
	}
	taken := make(map[uint64]bool, len(actors))
	orders := make([]Order, 0, len(actors))
	for _, actor := range actors {
		key, _ := keyOf(actor)
		i, ok := a.pickSlot(ctx, actor, slots)
		if !ok {
			continue
		}
		previous := ctx.internals.eval_errors
		weight := a.weightOf(ctx, actor, slots[i])
		if ctx.internals.eval_errors != previous || weight < 0 || math.IsNaN(weight) {
			if weight < 0 || math.IsNaN(weight) {
				ctx.internals.eval_errors++
			}
			continue
		}
		issued := a.issueVerbs(ctx, actor, &slots[i].aim, true)
		if len(issued) == 0 {
			continue
		}
		if a.prepend {
			taken[key.id] = true
		}
		if ctx.internals.target_queue_index != nil {
			if tk, ok := keyOf(slots[i].item); ok && tk.id != 0 {
				if ctx.internals.target_queue_index[key.id] == nil {
					ctx.internals.target_queue_index[key.id] = make(map[uint64]bool)
				}
				ctx.internals.target_queue_index[key.id][tk.id] = true
			}
		}
		slots[i].assigned += weight
		if slots[i].remaining != nil {
			*slots[i].remaining -= weight
			if *slots[i].remaining <= 0 {
				slots = slices.Delete(slots, i, i+1)
			}
		}
		orders = append(orders, issued...)
	}
	if a.prepend {
		for _, actor := range actors {
			key, _ := keyOf(actor)
			if taken[key.id] {
				if order, ok := ctx.requeue(actor); ok {
					orders = append(orders, order)
				}
			}
		}
	}
	return orders
}

func (a *selectAction) weightOf(ctx *issueContext, actor selItem, slot *targetSlot) float64 {
	if a.match == nil || a.match.weight == nil {
		return 1.0
	}
	restore := a.bind(ctx, actor, slot)
	defer restore()
	return a.match.weight.float(ctx.view, ctx.internals)
}

func (a *selectAction) bind(ctx *issueContext, actor selItem, slot *targetSlot) func() {
	restore_actor := ctx.internals.bindItem("actor", actor)
	restore_target := ctx.internals.bindItem("target", slot.item)
	return func() {
		restore_target()
		restore_actor()
	}
}
