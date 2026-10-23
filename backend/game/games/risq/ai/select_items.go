package ai

type tally struct {
	zone  map[ZoneRef]int
	space map[Coordinate]int
}

func newTally() tally {
	return tally{zone: map[ZoneRef]int{}, space: map[Coordinate]int{}}
}

func (t tally) add(loc ZoneRef) {
	t.zone[loc]++
	t.space[loc.Space]++
}

func (t tally) at(loc ZoneRef, whole_space bool) int {
	if whole_space {
		return t.space[loc.Space]
	}
	return t.zone[loc]
}

type census struct {
	invalid bool
	*worldIndex
	idle_units, idle_buildings map[uint64]bool
}

// The part of the census that does not depend on bucket or unit scoping, so it is shared while the view is unchanged
type worldIndex struct {
	revision                                                          uint64
	own_units, enemy_units, own_buildings, enemy_buildings, resources tally
	regions                                                           *regionMap
	spaces                                                            map[Coordinate]SpaceInfo
}

func (i *Internals) worldFor(view View) *worldIndex {
	if i.world == nil || i.world.revision != view.Revision() {
		i.world = newWorldIndex(view)
	}
	return i.world
}

func newCensus(view View, internals *Internals) *census {
	c := &census{worldIndex: internals.worldFor(view), idle_units: map[uint64]bool{}, idle_buildings: map[uint64]bool{}}
	for _, u := range view.IdleUnits() {
		c.idle_units[u.InternalID] = true
	}
	for _, b := range view.IdleBuildings() {
		c.idle_buildings[b.InternalID] = true
	}
	return c
}

func newWorldIndex(view View) *worldIndex {
	spaces := view.AllSpaces()
	c := &worldIndex{
		revision:  view.Revision(),
		own_units: newTally(), enemy_units: newTally(), own_buildings: newTally(),
		enemy_buildings: newTally(), resources: newTally(),
		regions: newRegionMap(view, spaces), spaces: map[Coordinate]SpaceInfo{},
	}
	for _, s := range spaces {
		c.spaces[s.Space] = s
	}
	for _, u := range view.AllUnits() {
		if u.PlayerID == view.PlayerID() {
			c.own_units.add(u.Location)
		} else {
			c.enemy_units.add(u.Location)
		}
	}
	for _, b := range view.AllBuildings() {
		if b.PlayerID == view.PlayerID() {
			c.own_buildings.add(b.Location)
		} else {
			c.enemy_buildings.add(b.Location)
		}
	}
	for _, category := range []ResourceCategory{ResourceFood, ResourceWood, ResourceStone, ResourceGold} {
		for _, r := range view.KnownResources(category) {
			c.resources.add(r.Location)
		}
	}
	return c
}

func (c *census) total(loc ZoneRef, whole_space bool) int {
	return c.own_units.at(loc, whole_space) + c.enemy_units.at(loc, whole_space) + c.own_buildings.at(loc, whole_space) +
		c.enemy_buildings.at(loc, whole_space) + c.resources.at(loc, whole_space)
}
