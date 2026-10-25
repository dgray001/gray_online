package ai

type assignKey struct {
	kind string
	key  itemKey
}

type assignPair struct {
	actor, target assignKey
}

type assignRecord struct {
	actor, target         loopItem
	actor_key, target_key assignKey
	weight                float64
	round                 int
}

type assignJob struct {
	action    *selectAction
	census    *census
	actors    []selItem
	slots     []*targetSlot
	records   []assignRecord
	loads     map[assignKey]float64
	pairs     map[assignPair]bool
	has_order map[assignKey]bool
	round     int
}

func newAssignJob(action *selectAction, cen *census) *assignJob {
	return &assignJob{action: action, census: cen, loads: map[assignKey]float64{}, pairs: map[assignPair]bool{}, has_order: map[assignKey]bool{}}
}

func assignKeyOf(item loopItem) assignKey {
	key, _ := keyOf(item)
	return assignKey{kind: kindOfItem(item), key: key}
}

func (i *Internals) storeAssignment(name string, job *assignJob) {
	if i.assignments == nil {
		i.assignments = make(map[string]*assignJob)
	}
	i.assignments[name] = job
}

func (j *assignJob) record(actor selItem, slot *targetSlot, weight float64) {
	actor_key := assignKeyOf(actor)
	j.records = append(j.records, assignRecord{
		actor: unwrapItem(actor), target: unwrapItem(slot.item),
		actor_key: actor_key, target_key: slot.assign_key, weight: weight, round: j.round,
	})
	j.loads[slot.assign_key] += weight
	j.pairs[assignPair{actor_key, slot.assign_key}] = true
	j.has_order[actor_key] = true
	slot.assigned += weight
	if slot.remaining != nil {
		*slot.remaining -= weight
	}
}
