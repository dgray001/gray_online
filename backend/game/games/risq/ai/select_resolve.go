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
	return compareCoordinates(a.loc.Zone, b.loc.Zone)
}

func (s *selectorSpec) resolve(view View, internals *Internals, cen *census) []selItem {
	if !s.validSets(internals, cen) {
		return nil
	}
	previous := internals.eval_errors
	anchors, ok := s.anchorSets(view, internals, cen)
	if !ok {
		cen.invalid = cen.invalid || internals.eval_errors != previous
		return nil
	}
	filter := s.newFilter(view, internals, cen, anchors)
	ranked := make([]rankedItem, 0)
	source, joined := s.candidates(view, internals)
	if !joined {
		source = s.source.items(view, internals)
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
			item.lets = &letState{specs: s.lets, busy: map[string]bool{}}
		}
		restore := internals.bindItem(s.role, item)
		key, _ := keyOf(item)
		previous := internals.eval_errors
		if filter.allows(item, key) {
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
	ranked = s.orderRanked(ranked, internals, skip+limit)[skip : skip+limit]
	items := make([]selItem, len(ranked))
	for i, r := range ranked {
		items[i] = r.item
		items[i].rank = r.primaryRank()
	}
	return items
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

func (s *selectorSpec) anchorSets(view View, internals *Internals, cen *census) ([][]ZoneRef, bool) {
	sets := make([][]ZoneRef, 0, len(s.near))
	for _, near := range s.near {
		if near.coordinate != nil {
			sets = append(sets, []ZoneRef{near.coordinate.resolve(view, internals)})
			continue
		}
		locations := make([]ZoneRef, 0)
		for _, item := range near.of.resolve(view, internals, cen) {
			if loc, _, located := locOf(item); located {
				locations = append(locations, loc)
			}
		}
		if len(locations) == 0 {
			return nil, false
		}
		sets = append(sets, locations)
	}
	return sets, true
}
