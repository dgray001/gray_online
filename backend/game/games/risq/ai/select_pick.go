package ai

import "slices"

func (a *selectAction) canIssue(ctx *issueContext, actor selItem, target *aim) bool {
	return slices.ContainsFunc(a.verbs, func(v verb) bool { return ctx.eligible(v, actor, target) })
}

func (i *Internals) pickTied(ties []int) int {
	if len(ties) > 1 && i.rng != nil {
		return ties[i.rng.Intn(len(ties))]
	}
	return ties[0]
}

func (a *selectAction) pickSlot(ctx *issueContext, actor selItem, slots []targetSlot, taken map[int]bool, round int) (int, bool) {
	if a.assign == nil {
		return 0, !taken[0] && a.canIssue(ctx, actor, &slots[0].aim)
	}
	var best, crowded []int
	best_score := 0.0
	for i := range slots {
		if taken[i] || !a.canIssue(ctx, actor, &slots[i].aim) {
			continue
		}
		restore := a.bind(ctx, actor, slots[i], round)
		previous := ctx.internals.eval_errors
		score := slots[i].item.rank + a.assign.unit_score.float(ctx.view, ctx.internals)
		full := a.assign.capacity != nil && slots[i].assigned >= a.assign.capacity.float(ctx.view, ctx.internals)
		restore()
		if ctx.internals.eval_errors != previous {
			continue
		}
		if full {
			if len(crowded) == 0 || slots[i].assigned < slots[crowded[0]].assigned {
				crowded = []int{i}
			} else if slots[i].assigned == slots[crowded[0]].assigned {
				crowded = append(crowded, i)
			}
			continue
		}
		if len(best) == 0 || score > best_score {
			best, best_score = []int{i}, score
		} else if score == best_score {
			best = append(best, i)
		}
	}
	if len(best) == 0 && round == 0 && a.assign.overflow == spreadOverflowRoundRobin {
		best = crowded
	}
	if len(best) == 0 {
		return 0, false
	}
	return ctx.internals.pickTied(best), true
}
