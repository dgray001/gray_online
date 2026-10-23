package ai

import (
	"fmt"
	"sort"
)

type unbucketedAction struct {
	inner Action
}

func (a *unbucketedAction) ToOrders(view View, internals *Internals) []Order {
	return a.inner.ToOrders(&unbucketedView{View: view, internals: internals}, internals)
}

// Runs any action with only the units its "unit_when" condition holds for; counters inside it can anchor on
// that unit ("from": "unit", or var(..._of_unit))
type unitWhenAction struct {
	when  Condition
	inner Action
}

func (a *unitWhenAction) ToOrders(view View, internals *Internals) []Order {
	return a.inner.ToOrders(&unitWhenView{View: view, internals: internals, when: a.when}, internals)
}

// Runs any action with only one bucket's members as eligible units
type inBucketAction struct {
	bucket string
	inner  Action
}

func (a *inBucketAction) ToOrders(view View, internals *Internals) []Order {
	b := internals.Buckets[a.bucket]
	if b == nil || len(b.Members) == 0 {
		return nil
	}
	return a.inner.ToOrders(&bucketView{View: view, members: b.Members}, internals)
}

type setBucketAction struct {
	bucket string
	size   amount
	task   Action
}

func (a *setBucketAction) ToOrders(view View, internals *Internals) []Order {
	b := internals.bucket(a.bucket)
	b.Desired = max(0, a.size.int(view, internals))
	b.Task = a.task
	if len(b.Members) > b.Desired {
		ids := make([]uint64, 0, len(b.Members))
		for id := range b.Members {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		for _, id := range ids[b.Desired:] {
			delete(b.Members, id)
		}
	}
	return nil
}

type fillBucketAction struct {
	filtered
	bucket   string
	eligible []OrderKind
}

func (a *fillBucketAction) ToOrders(view View, internals *Internals) []Order {
	b := internals.bucket(a.bucket)
	need := b.Desired - len(b.Members)
	candidates := a.filter.apply(eligibleUnits(view, a.eligible), anyUnit)
	if gather, ok := b.Task.(*gatherAction); ok {
		sort.SliceStable(candidates, func(i, j int) bool {
			return fillRank(candidates[i], gather.category) < fillRank(candidates[j], gather.category)
		})
	}
	for _, u := range candidates {
		if need <= 0 {
			break
		}
		if internals.isBucketed(u.InternalID) {
			continue
		}
		b.Members[u.InternalID] = true
		need--
	}
	return nil
}

// Prefers units already doing the bucket's gathering, then idle ones, so filling doesn't pull a unit off other work
func fillRank(u UnitView, category ResourceCategory) int {
	if u.CurrentOrder == nil {
		return 1
	}
	if u.CurrentOrder.TargetResource != nil && u.CurrentOrder.TargetResource.Category == category {
		return 0
	}
	return 2
}

type runBucketAction struct {
	bucket    string
	when_full bool
}

func (a *runBucketAction) ToOrders(view View, internals *Internals) []Order {
	b := internals.Buckets[a.bucket]
	if b == nil || b.Task == nil || len(b.Members) == 0 {
		return nil
	}
	if a.when_full && len(b.Members) < b.Desired {
		return nil
	}
	if internals.bucket_depth >= internals.bucketDepthLimit() {
		if !internals.bucket_depth_warn {
			internals.bucket_depth_warn = true
			fmt.Fprintln(internals.diagnostics, "ai: run_bucket", a.bucket, "skipped, nested past max_bucket_depth", internals.bucketDepthLimit())
		}
		return nil
	}
	internals.bucket_depth++
	defer func() { internals.bucket_depth-- }()
	return b.Task.ToOrders(&bucketView{View: view, members: b.Members}, internals)
}

type emptyBucketAction struct {
	bucket string
}

func (a *emptyBucketAction) ToOrders(_ View, internals *Internals) []Order {
	if b := internals.Buckets[a.bucket]; b != nil {
		b.Members = make(map[uint64]bool)
	}
	return nil
}

type drainBucketAction struct {
	to             string
	from           []string
	from_any       bool
	max            amount
	allow_reserved bool
}

func (a *drainBucketAction) ToOrders(view View, internals *Internals) []Order {
	limit := a.max.int(view, internals)
	to := internals.bucket(a.to)
	need := to.Desired - len(to.Members)
	if need <= 0 {
		return nil
	}
	source_names := a.from
	if a.from_any {
		source_names = make([]string, 0, len(internals.Buckets))
		for name := range internals.Buckets {
			if name != a.to {
				source_names = append(source_names, name)
			}
		}
		sort.Strings(source_names)
	}
	taken := 0
	for _, name := range source_names {
		from := internals.Buckets[name]
		if from == nil {
			continue
		}
		overflow := len(from.Members) - from.Desired
		ids := make([]uint64, 0, len(from.Members))
		for id := range from.Members {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		for _, id := range ids {
			if need <= 0 || (!a.allow_reserved && overflow <= 0) || (limit > 0 && taken >= limit) {
				break
			}
			delete(from.Members, id)
			to.Members[id] = true
			need--
			overflow--
			taken++
		}
		if need <= 0 || (limit > 0 && taken >= limit) {
			break
		}
	}
	return nil
}

type bucketView struct {
	View
	members map[uint64]bool
}

func (v *bucketView) filter(units []UnitView) []UnitView {
	out := make([]UnitView, 0, len(units))
	for _, u := range units {
		if v.members[u.InternalID] {
			out = append(out, u)
		}
	}
	return out
}

func (v *bucketView) ScopedUnits() []UnitView {
	return v.filter(v.View.ScopedUnits())
}

func (v *bucketView) IdleUnits() []UnitView {
	return v.filter(v.View.IdleUnits())
}

func (v *bucketView) EligibleUnits(kinds ...OrderKind) []UnitView {
	return v.filter(v.View.EligibleUnits(kinds...))
}

type unbucketedView struct {
	View
	internals *Internals
}

func (v *unbucketedView) filter(units []UnitView) []UnitView {
	out := make([]UnitView, 0, len(units))
	for _, u := range units {
		if !v.internals.isBucketed(u.InternalID) {
			out = append(out, u)
		}
	}
	return out
}

func (v *unbucketedView) ScopedUnits() []UnitView {
	return v.filter(v.View.ScopedUnits())
}

func (v *unbucketedView) IdleUnits() []UnitView {
	return v.filter(v.View.IdleUnits())
}

func (v *unbucketedView) EligibleUnits(kinds ...OrderKind) []UnitView {
	return v.filter(v.View.EligibleUnits(kinds...))
}

type unitWhenView struct {
	View
	internals *Internals
	when      Condition
}

func (v *unitWhenView) filter(units []UnitView) []UnitView {
	saved := v.internals.unit
	defer func() { v.internals.unit = saved }()
	out := make([]UnitView, 0, len(units))
	for _, u := range units {
		loc := u.Location
		v.internals.unit = &loc
		if matches, valid := evaluateCondition(v.when, v.View, v.internals); valid && matches {
			out = append(out, u)
		}
	}
	return out
}

func (v *unitWhenView) ScopedUnits() []UnitView {
	return v.filter(v.View.ScopedUnits())
}

func (v *unitWhenView) IdleUnits() []UnitView {
	return v.filter(v.View.IdleUnits())
}

func (v *unitWhenView) EligibleUnits(kinds ...OrderKind) []UnitView {
	return v.filter(v.View.EligibleUnits(kinds...))
}
