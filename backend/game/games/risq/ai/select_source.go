package ai

import (
	"cmp"
	"slices"
)

type keyedSource interface {
	itemsOf(keys []itemKey, view View, internals *Internals) ([]loopItem, bool)
}

func (s *selectorSpec) sourceItems(view View, internals *Internals) []loopItem {
	if keyed, ok := s.source.(keyedSource); ok {
		if keys, limited := s.memberKeys(internals); limited {
			if items, found := keyed.itemsOf(keys, view, internals); found {
				return items
			}
		}
	}
	return s.source.items(view, internals)
}

func (s *selectorSpec) memberKeys(internals *Internals) ([]itemKey, bool) {
	if s.in_set != "" {
		if set := internals.setNamed(s.in_set); set != nil {
			return set.keys, true
		}
		return nil, true
	}
	if s.bucket == "" || s.bucket == "none" {
		return nil, false
	}
	var keys []itemKey
	if bucket := internals.Buckets[s.bucket]; bucket != nil {
		for id := range bucket.Members {
			keys = append(keys, itemKey{id: id})
		}
	}
	return keys, true
}

func (s *unitSource) itemsOf(keys []itemKey, view View, internals *Internals) ([]loopItem, bool) {
	var lists [][]UnitView
	switch s.selector.relation {
	case relationOwn:
		lists = [][]UnitView{view.Units()}
	case relationEnemy:
		lists = [][]UnitView{view.VisibleEnemyUnits()}
	case relationAny:
		lists = [][]UnitView{view.Units(), view.VisibleEnemyUnits()}
	}
	ids := make([]uint64, len(keys))
	for i, key := range keys {
		ids[i] = key.id
	}
	slices.Sort(ids)
	items := make([]loopItem, 0, len(ids))
	for _, units := range lists {
		for _, id := range ids {
			at, found := slices.BinarySearchFunc(units, id, func(u UnitView, id uint64) int { return cmp.Compare(u.InternalID, id) })
			if !found {
				continue
			}
			if item, ok := s.item(view, internals, units[at]); ok {
				items = append(items, item)
			}
		}
	}
	return items, true
}

func (s *spaceSource) itemsOf(keys []itemKey, view View, internals *Internals) ([]loopItem, bool) {
	if !s.matchesEverySpace {
		return nil, false
	}
	infos := internals.worldFor(view).spaces
	coordinates := make([]Coordinate, 0, len(keys))
	for _, key := range keys {
		if _, exists := infos[key.loc.Space]; exists {
			coordinates = append(coordinates, key.loc.Space)
		}
	}
	slices.SortFunc(coordinates, compareCoordinates)
	return s.itemsAt(coordinates, view, internals), true
}

func (s *zoneSource) itemsOf(keys []itemKey, view View, internals *Internals) ([]loopItem, bool) {
	world := internals.worldFor(view)
	world.loadZones(view)
	positions := make([]int, 0, len(keys))
	for _, key := range keys {
		if position, exists := world.zone_position[key.loc]; exists {
			positions = append(positions, position)
		}
	}
	slices.Sort(positions)
	items := make([]loopItem, 0, len(positions))
	for _, position := range positions {
		if item, ok := s.item(view, internals, world.zones[position]); ok {
			items = append(items, item)
		}
	}
	return items, true
}
