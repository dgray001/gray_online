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

type idleSnapshotView interface {
	IdleSnapshot() (map[uint64]bool, map[uint64]bool)
}

// What the board holds. Orders cannot change it, so it is built once per decision and shared by every selector
type worldIndex struct {
	own_units, enemy_units, own_buildings, enemy_buildings, resources tally
	regions                                                           *regionMap
	spaces                                                            map[Coordinate]SpaceInfo
	zones                                                             []ZoneInfo
	zone_position                                                     map[ZoneRef]int
	own_by_space                                                      map[Coordinate]int
}

func (w *worldIndex) ownUnitsBySpace(view View) map[Coordinate]int {
	if w.own_by_space == nil {
		w.own_by_space = ownUnitsBySpace(view)
	}
	return w.own_by_space
}

func (i *Internals) worldFor(view View) *worldIndex {
	if i.world == nil {
		i.world = newWorldIndex(view)
	}
	return i.world
}

func (w *worldIndex) loadZones(view View) {
	if w.zone_position != nil {
		return
	}
	w.zones = view.AllZones()
	w.zone_position = make(map[ZoneRef]int, len(w.zones))
	for position, z := range w.zones {
		w.zone_position[z.Location] = position
	}
}

func newCensus(view View, internals *Internals) *census {
	c := &census{worldIndex: internals.worldFor(view)}
	if snapshot, ok := view.(idleSnapshotView); ok {
		c.idle_units, c.idle_buildings = snapshot.IdleSnapshot()
		return c
	}
	c.idle_units, c.idle_buildings = map[uint64]bool{}, map[uint64]bool{}
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
