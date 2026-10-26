package ai

import "slices"

type itemFilter struct {
	spec                               *selectorSpec
	view                               View
	internals                          *Internals
	in_set, not_in_set                 map[itemKey]bool
	members                            map[uint64]bool
	places                             map[Coordinate]bool
	has_in_set, has_bucket, has_places bool
	region_keys                        map[itemKey]bool
	has_region_set                     bool
	near_clauses                       []nearClause
	scope                              func(selItem) bool
}

func (s *selectorSpec) newFilter(view View, internals *Internals, cen *census, clauses []nearClause) itemFilter {
	f := itemFilter{spec: s, view: view, internals: internals, near_clauses: clauses}
	if s.role == "actor" {
		f.scope = actorScope(view, s.kind)
	}
	if s.in_set != "" {
		f.has_in_set, f.in_set = true, setKeys(internals.setNamed(s.in_set))
	}
	if s.not_in_set != "" {
		f.not_in_set = setKeys(internals.setNamed(s.not_in_set))
	}
	if s.region_in_set != "" {
		f.has_region_set, f.region_keys = true, setKeys(internals.setNamed(s.region_in_set))
	}
	if s.bucket != "" {
		f.has_bucket, f.members = true, map[uint64]bool{}
		if s.bucket == "none" {
			f.members = internals.bucketMembers()
		} else if b := internals.Buckets[s.bucket]; b != nil {
			f.members = b.Members
		}
	}
	if s.at != nil {
		f.has_places, f.places = true, map[Coordinate]bool{}
		for _, item := range s.at.resolve(view, internals, cen) {
			if loc, _, located := locOf(item); located {
				f.places[loc.Space] = true
			}
		}
	}
	return f
}

func actorScope(view View, kind string) func(selItem) bool {
	switch kind {
	case "units":
		return func(item selItem) bool { return unwrapItem(item).(unitItem).unit.PlayerID == view.PlayerID() }
	case "buildings":
		return func(item selItem) bool { return unwrapItem(item).(buildingItem).building.PlayerID == view.PlayerID() }
	}
	return nil
}

func setKeys(set *entitySet) map[itemKey]bool {
	if set == nil {
		return nil
	}
	return set.lookup
}

func (f itemFilter) allows(item selItem, key itemKey) bool {
	if source, ok := f.spec.source.(*orderSource); ok && !source.allows(unwrapItem(item).(orderItem).entry.SubjectUID, f.internals) {
		return false
	}
	loc, _, located := locOf(item)
	switch {
	case f.scope != nil && !f.scope(item):
		return false
	case f.has_in_set && !f.in_set[key], f.not_in_set[key]:
		return false
	case f.has_bucket && f.members[key.id] == (f.spec.bucket == "none"):
		return false
	case f.has_places && (!located || !f.places[loc.Space]):
		return false
	case f.has_region_set && !f.regionAllowed(item, loc):
		return false
	case !f.nearAll(item, loc):
		return false
	case !f.spec.offers(item):
		return false
	}
	for _, filter := range f.spec.in {
		if value, ok := item.field(filter.field); !ok || !filter.allows(value) {
			return false
		}
	}
	for _, condition := range f.spec.conditions {
		if condition.float(f.view, f.internals) <= 0 {
			return false
		}
	}
	return true
}

func (f itemFilter) regionAllowed(item selItem, loc ZoneRef) bool {
	stat, inside := item.census.regions.at(loc.Space)
	return inside && f.region_keys[itemKey{name: stat.info.Name}]
}

func (f itemFilter) nearAll(item selItem, loc ZoneRef) bool {
	for i, near := range f.spec.near {
		clause := f.near_clauses[i]
		max_d := float64(clause.bound)
		if !clause.hoisted {
			max_d = float64(near.distance.int(f.view, f.internals))
		}
		if clause.reach != nil {
			d, reached := clause.reach[loc.Space]
			if !reached || float64(d) > max_d {
				return false
			}
			continue
		}
		within := slices.ContainsFunc(clause.locations, func(anchor ZoneRef) bool {
			return item.measure(loc, anchor, near.zone_steps) <= max_d
		})
		if !within {
			return false
		}
	}
	return true
}
