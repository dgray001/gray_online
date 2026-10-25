package ai

import (
	"slices"
	"sort"
)

type rankedItem struct {
	item selItem
	key  itemKey
	rank []float64
}

func (r rankedItem) primaryRank() float64 {
	if len(r.rank) == 0 {
		return 0
	}
	return r.rank[0]
}

func compareKeys(a itemKey, b itemKey) int {
	if a.name != b.name {
		if a.name < b.name {
			return -1
		}
		return 1
	}
	if a.id != b.id {
		if a.id < b.id {
			return -1
		}
		return 1
	}
	if c := compareCoordinates(a.loc.Space, b.loc.Space); c != 0 {
		return c
	}
	if a.position != b.position {
		if a.position < b.position {
			return -1
		}
		return 1
	}
	return compareCoordinates(a.loc.Zone, b.loc.Zone)
}

func (s *selectorSpec) resolve(view View, internals *Internals, cen *census) []selItem {
	if !s.validSets(internals, cen) {
		return nil
	}
	previous := internals.eval_errors
	clauses, ok := s.anchorSets(view, internals, cen)
	if !ok {
		cen.invalid = cen.invalid || internals.eval_errors != previous
		return nil
	}
	filter := s.newFilter(view, internals, cen, clauses)
	ranked := make([]rankedItem, 0)
	source, joined := s.candidates(view, internals)
	if !joined {
		source = s.sourceItems(view, internals)
	}
	if internals.eval_errors != previous {
		cen.invalid = true
		return nil
	}
	for _, base := range source {
		if !internals.takeSelectStep() {
			return nil
		}
		item := selItem{loopItem: base, view: view, internals: internals, census: cen, role: s.role}
		if len(s.lets) > 0 {
			item.lets = &letState{specs: s.lets, static: s.role == "target" || s.role == "actor"}
		}
		restore := internals.bindItem(s.role, item)
		if item.lets != nil {
			item.lets.cache_depth = internals.binding_depth
		}
		key, _ := keyOf(item)
		previous := internals.eval_errors
		if filter.allows(item, key) {
			if s.initial != nil {
				initial, valid := s.initialLoad(view, internals, previous)
				if !valid {
					restore()
					continue
				}
				item.initial = &initial
				restore()
				restore = internals.bindItem(s.role, item)
			}
			if s.capacity != nil && s.role == "target" {
				c := s.capacity.float(view, internals)
				if c <= 0 || internals.eval_errors != previous {
					restore()
					continue
				}
				item.capacity = &c
			}
			rank := s.rankOf(view, internals)
			if internals.eval_errors == previous {
				ranked = append(ranked, rankedItem{item: item, key: key, rank: rank})
			}
		}
		restore()
	}
	if cen.invalid {
		return nil
	}
	previous = internals.eval_errors
	skip, limit := s.skipAndLimit(view, internals, len(ranked))
	if internals.eval_errors != previous {
		cen.invalid = true
		return nil
	}
	if skip == 0 && limit == 1 && len(ranked) > 1 {
		ranked = []rankedItem{bestRanked(ranked, internals)}
	} else {
		ranked = s.orderRanked(ranked, internals, skip+limit)[skip : skip+limit]
	}
	items := make([]selItem, len(ranked))
	for i, r := range ranked {
		items[i] = r.item
		items[i].rank = r.primaryRank()
		items[i].rank_keys = r.rank
		if len(s.lets) > 0 && s.role != "target" && s.role != "actor" {
			items[i].lets = &letState{specs: s.lets}
		}
	}
	return items
}

func (s *selectorSpec) initialLoad(view View, internals *Internals, previous uint64) (float64, bool) {
	load := s.initial.float(view, internals)
	if internals.eval_errors != previous {
		return 0, false
	}
	if load < 0 {
		internals.warnOnce("initial_assigned", "ai: initial_assigned must not be negative")
		internals.eval_errors++
		if cen := internals.select_census; cen != nil {
			cen.invalid = true
		}
		return 0, false
	}
	return load, true
}

func bestRanked(ranked []rankedItem, internals *Internals) rankedItem {
	ties := []rankedItem{ranked[0]}
	for _, r := range ranked[1:] {
		switch order := slices.Compare(r.rank, ties[0].rank); {
		case order > 0:
			ties = append(ties[:0], r)
		case order == 0:
			ties = append(ties, r)
		}
	}
	if internals.rng == nil {
		return slices.MinFunc(ties, func(a, b rankedItem) int { return compareKeys(a.key, b.key) })
	}
	if len(ties) == 1 {
		return ties[0]
	}
	return ties[internals.rng.Intn(len(ties))]
}

func (s *selectorSpec) skipAndLimit(view View, internals *Internals, total int) (int, int) {
	skip := 0
	if s.skip != nil {
		skip = min(total, max(0, s.skip.int(view, internals)))
	}
	limit := total - skip
	if s.limit != nil {
		limit = min(total-skip, max(0, s.limit.int(view, internals)))
	}
	return skip, limit
}

func (s *selectorSpec) orderRanked(ranked []rankedItem, internals *Internals, need int) []rankedItem {
	if len(s.rank) == 0 && s.limit == nil && s.skip == nil {
		return ranked
	}
	orderDrawingTies(internals, ranked, need,
		func(a, b rankedItem) int { return slices.Compare(a.rank, b.rank) },
		func(a, b rankedItem) bool { return compareKeys(a.key, b.key) < 0 })
	return ranked
}

func orderDrawingTies[T any](internals *Internals, items []T, need int, compare func(a, b T) int, fallback func(a, b T) bool) {
	sort.SliceStable(items, func(a, b int) bool {
		if order := compare(items[a], items[b]); order != 0 {
			return order > 0
		}
		return internals.rng == nil && fallback(items[a], items[b])
	})
	for start := 0; start < need && internals.rng != nil; {
		end := start + 1
		for end < len(items) && compare(items[end], items[start]) == 0 {
			end++
		}
		for p := start; p < min(end, need) && end-p > 1; p++ {
			j := p + internals.rng.Intn(end-p)
			items[p], items[j] = items[j], items[p]
		}
		start = end
	}
}

func (s *selectorSpec) rankOf(view View, internals *Internals) []float64 {
	ranks := make([]float64, len(s.rank))
	for i, rank := range s.rank {
		ranks[i] = rank.float(view, internals)
	}
	return ranks
}

func (s *selectorSpec) anchorSets(view View, internals *Internals, cen *census) ([]nearClause, bool) {
	clauses := make([]nearClause, 0, len(s.near))
	for _, near := range s.near {
		locations := s.resolveNearLocations(near, view, internals, cen)
		if len(locations) == 0 {
			return nil, false
		}
		var reach map[Coordinate]int
		if near.through != nil {
			reach = computeThroughReach(near, locations, view, internals, cen)
			if reach == nil {
				return nil, false
			}
		}
		clause := nearClause{locations: locations, reach: reach}
		if near.distance.independentOf(s.role) {
			errors := internals.eval_errors
			clause.bound, clause.hoisted = near.distance.int(view, internals), true
			if internals.eval_errors != errors {
				return nil, false
			}
		}
		if !near.zone_steps && reach == nil {
			clause.locations = distinctSpaces(locations)
		}
		clauses = append(clauses, clause)
	}
	return clauses, true
}

func distinctSpaces(locations []ZoneRef) []ZoneRef {
	seen := make(map[Coordinate]bool, len(locations))
	distinct := make([]ZoneRef, 0, len(locations))
	for _, loc := range locations {
		if !seen[loc.Space] {
			seen[loc.Space] = true
			distinct = append(distinct, ZoneRef{Space: loc.Space})
		}
	}
	return distinct
}

func (s *selectorSpec) resolveNearLocations(near *nearSpec, view View, internals *Internals, cen *census) []ZoneRef {
	if near.coordinate != nil {
		return []ZoneRef{near.coordinate.resolve(view, internals)}
	}
	locations := make([]ZoneRef, 0)
	for _, item := range near.of.resolve(view, internals, cen) {
		if loc, _, located := locOf(item); located {
			locations = append(locations, loc)
		}
	}
	return locations
}

func computeThroughReach(near *nearSpec, locations []ZoneRef, view View, internals *Internals, cen *census) map[Coordinate]int {
	previous := internals.eval_errors
	allowed := map[Coordinate]bool{}
	for _, item := range near.through.resolve(view, internals, cen) {
		if loc, _, located := locOf(item); located {
			allowed[loc.Space] = true
		}
	}
	if cen.invalid || internals.eval_errors != previous {
		cen.invalid = true
		return nil
	}
	reach := map[Coordinate]int{}
	queue := make([]Coordinate, 0, len(locations))
	for _, loc := range locations {
		if _, seen := reach[loc.Space]; !seen {
			reach[loc.Space] = 0
			queue = append(queue, loc.Space)
		}
	}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		next_d := reach[curr] + 1
		for _, neighbor := range view.SpaceNeighbors(curr) {
			if !allowed[neighbor] {
				continue
			}
			if _, seen := reach[neighbor]; !seen {
				reach[neighbor] = next_d
				queue = append(queue, neighbor)
			}
		}
	}
	return reach
}
