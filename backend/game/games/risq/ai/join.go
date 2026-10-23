package ai

import (
	"math"
	"strings"

	"github.com/dgray001/gray_online/util"
)

const maxJoinParts = 5

type joinKey [maxJoinParts]float64

// A source whose items do not depend on bindings, buckets or variables, only on the view
type staticSource interface {
	static() bool
}

func (s selector) static() bool          { return s.player == nil && !s.near.has_within }
func (s *unitSource) static() bool       { return s.selector.static() }
func (s *buildingSource) static() bool   { return s.selector.static() }
func (s *attackableSource) static() bool { return s.selector.static() }
func (s *resourceSource) static() bool   { return !s.near.has_within }
func (s *gatherableSource) static() bool { return s.resources.static() }
func (s *foundationSource) static() bool { return true }

// The `==` factors of a selector's conditions that compare one of the item's own fields with something that
// does not read the item, so matching items can be found by key instead of by testing every item
type joinPlan struct {
	inner []string
	outer []amount
}

// Items of one source keyed by the plan's inner expressions, valid for one view revision
type joinIndex struct {
	revision uint64
	items    []loopItem
	buckets  map[joinKey][]loopItem
	unusable bool
}

func (s *selectorSpec) itemSide(text string) bool {
	reads, ok := util.AnalyzeExpr(text)
	if !ok || reads.Impure || len(reads.Bare) > 0 || len(reads.Vars) == 0 {
		return false
	}
	for _, name := range reads.Vars {
		role, field, dotted := strings.Cut(name, ".")
		if !dotted || role != s.role || !hasItemField(s.kind, field) {
			return false
		}
	}
	return true
}

func (s *selectorSpec) otherSide(text string) (amount, bool) {
	reads, ok := util.AnalyzeExpr(text)
	if !ok || reads.Impure {
		return amount{}, false
	}
	for _, name := range reads.Vars {
		if strings.HasPrefix(name, s.role+".") {
			return amount{}, false
		}
	}
	return amount{expr: text, reads_vars: len(reads.Bare) > 0}, true
}

func (s *selectorSpec) joinPlanFor(internals *Internals) *joinPlan {
	if s.join_planned {
		return s.join
	}
	s.join_planned = true
	source, ok := s.source.(staticSource)
	if internals.random_used || !ok || !source.static() {
		return nil
	}
	plan := &joinPlan{}
	for _, condition := range s.conditions {
		for _, factor := range util.EqualityFactors(condition.expr) {
			for _, sides := range [][2]string{{factor.Left, factor.Right}, {factor.Right, factor.Left}} {
				if outer, ok := s.otherSide(sides[1]); ok && s.itemSide(sides[0]) && len(plan.inner) < maxJoinParts {
					plan.inner, plan.outer = append(plan.inner, sides[0]), append(plan.outer, outer)
					break
				}
			}
		}
	}
	if len(plan.inner) > 0 {
		s.join = plan
	}
	return s.join
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func (p *joinPlan) itemKey(item loopItem) (joinKey, bool) {
	var key joinKey
	for i, text := range p.inner {
		value, err := util.EvalExprWithLookupAndRng(text, nil, func(name string) (float64, error) {
			_, field, _ := strings.Cut(name, ".")
			value, _ := item.field(field)
			return value, nil
		}, nil)
		if err != nil || !finite(value) {
			return key, false
		}
		key[i] = value
	}
	return key, true
}

func (p *joinPlan) outerKey(view View, internals *Internals) (joinKey, bool) {
	var key joinKey
	for i, part := range p.outer {
		value, err := part.resolve(view, internals)
		if err != nil {
			return key, false
		}
		key[i] = value
	}
	return key, true
}

func (s *selectorSpec) buildJoinIndex(plan *joinPlan, view View, internals *Internals) *joinIndex {
	index := &joinIndex{revision: view.Revision(), buckets: map[joinKey][]loopItem{}}
	previous := internals.eval_errors
	index.items = s.source.items(view, internals)
	if internals.eval_errors != previous {
		index.unusable = true
		return index
	}
	for _, item := range index.items {
		if !internals.takeSelectStep() {
			return nil
		}
		key, ok := plan.itemKey(item)
		if !ok {
			index.unusable = true
			return index
		}
		index.buckets[key] = append(index.buckets[key], item)
	}
	return index
}

// The items worth testing against the filter: only those matching the join key when the selector has one.
// Testing the rest could not succeed because the equality factors are part of the conditions.
func (s *selectorSpec) candidates(view View, internals *Internals) ([]loopItem, bool) {
	plan := s.joinPlanFor(internals)
	if plan == nil {
		return nil, false
	}
	index := internals.joins[s]
	if index == nil || index.revision != view.Revision() {
		if index = s.buildJoinIndex(plan, view, internals); index == nil {
			return nil, false
		}
		if internals.joins == nil {
			internals.joins = map[*selectorSpec]*joinIndex{}
		}
		internals.joins[s] = index
	}
	if index.unusable {
		return nil, false
	}
	errors := internals.eval_errors
	key, ok := plan.outerKey(view, internals)
	internals.eval_errors = errors
	if !ok {
		return index.items, true
	}
	return index.buckets[key], true
}
