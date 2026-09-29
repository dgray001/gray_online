package ai

import (
	"fmt"
	"os"
	"sort"
)

type unbucketedAction struct {
	inner Action
}

func (a *unbucketedAction) ToOrders(view View, internals *Internals) []Order {
	return a.inner.ToOrders(&unbucketedView{View: view, internals: internals}, internals)
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
			fmt.Fprintln(os.Stderr, "ai: run_bucket", a.bucket, "skipped, nested past max_bucket_depth", internals.bucketDepthLimit())
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
	to       string
	from     []string
	from_any bool
	max      amount
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
			if need <= 0 || overflow <= 0 || (limit > 0 && taken >= limit) {
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
