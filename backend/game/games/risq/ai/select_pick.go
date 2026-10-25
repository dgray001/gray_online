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

func (a *selectAction) scoreOf(ctx *issueContext, slot *targetSlot) []float64 {
	if a.match != nil && len(a.match.score) > 0 {
		score := make([]float64, len(a.match.score))
		for sIdx, sAmt := range a.match.score {
			score[sIdx] = sAmt.float(ctx.view, ctx.internals)
		}
		return score
	}
	if len(slot.item.rank_keys) > 0 {
		return slot.item.rank_keys
	}
	return []float64{slot.item.rank}
}

func (a *selectAction) evalCandidate(ctx *issueContext, actor selItem, slot *targetSlot) ([]float64, bool) {
	restore := a.bind(ctx, actor, slot)
	defer restore()
	previous := ctx.internals.eval_errors
	if a.match.where != nil && !a.match.where.Evaluate(ctx.view, ctx.internals) {
		return nil, false
	}
	if ctx.internals.eval_errors != previous {
		return nil, false
	}
	score := a.scoreOf(ctx, slot)
	if ctx.internals.eval_errors != previous {
		return nil, false
	}
	return score, true
}
func (a *selectAction) pickSlot(ctx *issueContext, actor selItem, slots []*targetSlot) (int, bool) {
	if a.match == nil {
		for i, slot := range slots {
			if !ctx.internals.takeSelectStep() {
				return 0, false
			}
			if ctx.excluded != nil && ctx.excluded(slot) {
				continue
			}
			if (slot.capacity == nil || *slot.remaining > 0) && a.canIssue(ctx, actor, &slot.aim) {
				return i, true
			}
		}
		return 0, false
	}
	var best []int
	var bestScore []float64
	for i, slot := range slots {
		if !ctx.internals.takeSelectStep() {
			return 0, false
		}
		if slot.capacity != nil && *slot.remaining <= 0 {
			continue
		}
		if ctx.excluded != nil && ctx.excluded(slot) {
			continue
		}
		score, ok := a.evalCandidate(ctx, actor, slot)
		if !ok || !a.canIssue(ctx, actor, &slot.aim) {
			continue
		}
		if len(best) == 0 || slices.Compare(score, bestScore) > 0 {
			best, bestScore = []int{i}, score
		} else if slices.Compare(score, bestScore) == 0 {
			best = append(best, i)
		}
	}
	if len(best) == 0 {
		return 0, false
	}
	return ctx.internals.pickTied(best), true
}
