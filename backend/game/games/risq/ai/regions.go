package ai

type regionStat struct {
	index                 int
	info                  RegionInfo
	owned, enemy, unowned int
}

type regionMap struct {
	stats []regionStat
	of    map[Coordinate]int
}

func newRegionMap(view View, spaces []SpaceInfo) *regionMap {
	m := &regionMap{of: map[Coordinate]int{}}
	owners := make(map[Coordinate]int)
	for _, s := range spaces {
		owners[s.Space] = s.Owner
	}
	for i, info := range view.AllRegions() {
		stat := regionStat{index: i, info: info}
		for _, space := range info.Spaces {
			m.of[space] = i
			owner, known := owners[space]
			switch {
			case !known || owner < 0:
				stat.unowned++
			case owner == view.PlayerID():
				stat.owned++
			default:
				stat.enemy++
			}
		}
		m.stats = append(m.stats, stat)
	}
	return m
}

func (m *regionMap) at(space Coordinate) (regionStat, bool) {
	index, ok := m.of[space]
	if !ok {
		return regionStat{index: -1, info: RegionInfo{Owner: -1}}, false
	}
	return m.stats[index], true
}
