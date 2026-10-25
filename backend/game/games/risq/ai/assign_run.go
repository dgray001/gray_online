package ai

import (
	"fmt"
	"math"
)

func (a *selectAction) startAssignment(ctx *issueContext, job *assignJob, actors []selItem, targets []selItem) []Order {
	job.slots = a.slotsFor(ctx, targets)
	for _, actor := range actors {
		if !preEvalLets(actor, ctx.internals) {
			if actor.census.invalid {
				return nil
			}
			continue
		}
		job.actors = append(job.actors, actor)
	}
	return job.runRound(ctx)
}

func (a *selectAction) carryRound(view View, internals *Internals) []Order {
	job := internals.assignments[a.carry]
	if job == nil {
		internals.warnOnce("carry:"+a.carry, fmt.Sprintf("ai: carry names no assignment %q", a.carry))
		internals.eval_errors++
		return nil
	}
	previous := internals.select_census
	internals.select_census = job.census
	defer func() { internals.select_census = previous }()
	job.round++
	return job.runRound(&issueContext{view: view, internals: internals})
}

func (j *assignJob) runRound(ctx *issueContext) []Order {
	a := j.action
	defer saveScoringContext(ctx.internals).restore(ctx.internals)
	ctx.internals.assign_round = float64(j.round)
	defer func() { ctx.excluded = nil }()
	orders := make([]Order, 0, len(j.actors))
	for _, actor := range j.actors {
		actor_key := assignKeyOf(actor)
		ctx.excluded = func(slot *targetSlot) bool { return j.pairs[assignPair{actor_key, slot.assign_key}] }
		i, ok := a.pickSlot(ctx, actor, j.slots)
		if !ok {
			continue
		}
		previous := ctx.internals.eval_errors
		weight := a.weightOf(ctx, actor, j.slots[i])
		if ctx.internals.eval_errors != previous || weight < 0 || math.IsNaN(weight) {
			if weight < 0 || math.IsNaN(weight) {
				ctx.internals.eval_errors++
			}
			continue
		}
		issued := a.issueVerbs(ctx, actor, &j.slots[i].aim, !j.has_order[actor_key])
		if len(issued) == 0 {
			continue
		}
		noteQueuedTarget(ctx.internals, actor, j.slots[i].item)
		j.record(actor, j.slots[i], weight)
		orders = append(orders, issued...)
	}
	return orders
}

func noteQueuedTarget(internals *Internals, actor selItem, target selItem) {
	if internals.target_queue_index == nil {
		return
	}
	actor_key, target_key := assignKeyOf(actor).key, assignKeyOf(target).key
	if target_key.id == 0 {
		return
	}
	if internals.target_queue_index[actor_key.id] == nil {
		internals.target_queue_index[actor_key.id] = make(map[uint64]bool)
	}
	internals.target_queue_index[actor_key.id][target_key.id] = true
}
