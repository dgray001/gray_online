package ai

import "fmt"

type entitySet struct {
	kind string
	keys []itemKey
}

func (s *selectorSpec) validSets(internals *Internals, cen *census) bool {
	for _, filter := range []struct{ name, kind string }{{s.in_set, s.kind}, {s.not_in_set, s.kind}, {s.region_in_set, "regions"}} {
		if filter.name == "" {
			continue
		}
		if set := internals.setNamed(filter.name); set != nil && set.kind != filter.kind {
			internals.warnOnce("set_kind:"+filter.name+":"+filter.kind, fmt.Sprintf("ai: set %q contains %s, expected %s", filter.name, set.kind, filter.kind))
			cen.invalid = true
			return false
		}
	}
	return !cen.invalid
}

type itemKey struct {
	id   uint64
	name string
	loc  ZoneRef
}

func keyOf(item loopItem) (itemKey, bool) {
	switch it := unwrapItem(item).(type) {
	case unitItem:
		return itemKey{id: unitUID(it.unit.InternalID)}, true
	case buildingItem:
		return itemKey{id: buildingUID(it.building.InternalID)}, true
	case resourceItem:
		return itemKey{id: it.resource.uid()}, true
	case spaceItem:
		return itemKey{loc: ZoneRef{Space: it.loc.Space}}, true
	case zoneItem:
		return itemKey{loc: it.loc}, true
	case regionItem:
		return itemKey{name: it.stat.info.Name}, true
	case foundationItem:
		return itemKey{loc: it.loc}, true
	case gatherableItem:
		return itemKey{id: it.node.uid()}, true
	case attackableItem:
		return itemKey{id: it.uid}, true
	}
	return itemKey{}, false
}

func (i *Internals) setNamed(name string) *entitySet {
	if set, ok := i.turn_sets[name]; ok {
		return set
	}
	return i.sets[name]
}

func (i *Internals) storeSet(name string, set *entitySet, persist bool) {
	target, other := &i.turn_sets, i.sets
	if persist {
		target, other = &i.sets, i.turn_sets
	}
	delete(other, name)
	if *target == nil {
		*target = make(map[string]*entitySet)
	}
	(*target)[name] = set
}
