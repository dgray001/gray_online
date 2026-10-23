package ai

type homePoint struct {
	loc   ZoneRef
	known bool
}

func newHomePoint(view View) homePoint {
	loc, known := homeLocation(view)
	return homePoint{loc: loc, known: known}
}

func (h homePoint) placeOf(view View, loc ZoneRef) place {
	p := place{loc: loc}
	if h.known {
		p.distance_home = view.SpaceDistance(h.loc.Space, loc.Space)
	}
	return p
}

type legacyHomeSource struct{ loopSource }

func (s legacyHomeSource) items(view View, internals *Internals) []loopItem {
	items := s.loopSource.items(view, internals)
	home := newHomePoint(view)
	for i, item := range items {
		loc, _, _ := locOf(item)
		items[i] = withLegacyPlace(item, home.placeOf(view, loc))
	}
	return items
}

func withLegacyPlace(item loopItem, p place) loopItem {
	switch it := item.(type) {
	case spaceItem:
		it.place = p
		return it
	case unitItem:
		it.place = p
		return it
	case buildingItem:
		it.place = p
		return it
	case resourceItem:
		it.place = p
		return it
	case zoneItem:
		it.place = p
		return it
	}
	return item
}
