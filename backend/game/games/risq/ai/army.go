package ai

// armyAction runs the whole military as two groups: defenders (everything not in the assault) hold home and
// converge on threats near any of our buildings, while an assault group of the chosen unit types launches once
// enough of them are gathered at home, stays together, and falls back when too few survive.
type armyAction struct {
	assault_ids   map[uint32]bool
	launch        int
	retreat       int
	defend_radius int
	// launches an all-units strike right after a big enemy army was broken, if we still have this many (0 disables)
	strike int
}

type armyState struct {
	// biggest visible enemy army in the last ~15 turns, for spotting a broken enemy
	peak      int
	peak_turn int
	assault   bool
	members   map[uint64]bool
	focus     *ZoneRef
	// last seen enemy buildings, kept until an assault unit stands there and finds them gone
	known map[uint64]ZoneRef
}

func (a *armyAction) inAssaultGroup(u UnitView) bool {
	return len(a.assault_ids) == 0 || a.assault_ids[u.UnitID]
}

func (a *armyAction) ToOrders(view View, internals *Internals) []Order {
	st := &internals.army
	if st.members == nil {
		st.members = make(map[uint64]bool)
	}
	orders := make([]Order, 0)
	mil := make([]UnitView, 0)
	alive := make(map[uint64]bool)
	for _, u := range view.Units() {
		if isMilitary(u) {
			alive[u.InternalID] = true
			if u.GarrisonedIn == nil && !internals.isBucketed(u.InternalID) {
				mil = append(mil, u)
			}
		}
	}
	for id := range st.members {
		if !alive[id] {
			delete(st.members, id)
		}
	}
	home, has_home := homeLocation(view)
	if len(mil) == 0 || !has_home {
		return orders
	}

	enemy_army := 0
	for _, e := range view.VisibleEnemyUnits() {
		if e.Kind == UnitMilitary && axialDistance(home.Space, e.Location.Space) <= a.defend_radius+1 {
			enemy_army++
		}
	}
	turn := view.TurnNumber()
	if enemy_army >= st.peak || turn-st.peak_turn > 15 {
		st.peak, st.peak_turn = enemy_army, turn
	}
	enemy_broken := a.strike > 0 && st.peak >= 15 && enemy_army <= st.peak/3

	// launch / retreat decision
	if !st.assault && enemy_broken && len(mil) >= a.strike {
		st.assault = true
		st.members = make(map[uint64]bool, len(mil))
		for _, u := range mil {
			st.members[u.InternalID] = true
		}
		st.peak = 0
	} else if !st.assault {
		gathered := make([]UnitView, 0)
		for _, u := range mil {
			if a.inAssaultGroup(u) && axialDistance(u.Location.Space, home.Space) <= 2 {
				gathered = append(gathered, u)
			}
		}
		if len(gathered) >= a.launch {
			st.assault = true
			st.members = make(map[uint64]bool, len(gathered))
			for _, u := range gathered {
				st.members[u.InternalID] = true
			}
		}
	} else if len(st.members) <= a.retreat {
		st.assault = false
		st.members = make(map[uint64]bool)
		st.focus = nil
	}

	enemies := view.VisibleEnemyUnits()
	threats := make([]UnitView, 0)
	for _, e := range enemies {
		if axialDistance(home.Space, e.Location.Space) <= a.defend_radius {
			threats = append(threats, e)
		}
	}

	members := make([]UnitView, 0)
	defenders := make([]UnitView, 0)
	for _, u := range mil {
		if st.members[u.InternalID] {
			members = append(members, u)
		} else {
			defenders = append(defenders, u)
		}
	}
	defender_targets := spreadTargets(defenders, threats, 2)
	for _, u := range defenders {
		if t, ok := defender_targets[u.InternalID]; ok {
			orders = append(orders, view.AttackUnitOrder(u, t, true))
		} else if len(threats) == 0 && u.CurrentOrder == nil && axialDistance(u.Location.Space, home.Space) >= 1 {
			orders = append(orders, view.MoveOrder(u, home, true))
		}
	}
	if len(members) == 0 {
		return orders
	}
	if !st.assault {
		return orders
	}
	return append(orders, a.assaultOrders(view, st, members, enemies)...)
}

func (a *armyAction) assaultOrders(view View, st *armyState, members []UnitView, enemies []UnitView) []Order {
	orders := make([]Order, 0, len(members))
	// the member closest to everyone else is the group's anchor
	anchor, best := members[0], -1
	for _, u := range members {
		sum := 0
		for _, v := range members {
			sum += axialDistance(u.Location.Space, v.Location.Space)
		}
		if best < 0 || sum < best {
			anchor, best = u, sum
		}
	}
	enemy_buildings := view.VisibleEnemyBuildings()
	if st.known == nil {
		st.known = make(map[uint64]ZoneRef)
	}
	visible := make(map[uint64]bool, len(enemy_buildings))
	for _, b := range enemy_buildings {
		visible[b.InternalID] = true
		st.known[b.InternalID] = b.Location
	}
	for id, loc := range st.known {
		if visible[id] {
			continue
		}
		for _, u := range members {
			if u.Location.Space == loc.Space {
				delete(st.known, id)
				break
			}
		}
	}
	if b, ok := nearestBuilding(view, anchor.Location, enemy_buildings); ok {
		focus := b.Location
		st.focus = &focus
	} else if len(st.known) > 0 {
		remembered := make([]ZoneRef, 0, len(st.known))
		for _, loc := range st.known {
			remembered = append(remembered, loc)
		}
		if loc, ok := nearestZone(view, anchor.Location, remembered); ok {
			st.focus = &loc
		}
	} else {
		st.focus = nil
	}
	// units standing on a building's zone can't be reached in melee, so they're only dealt with by razing the building
	building_zones := make(map[ZoneRef]bool, len(enemy_buildings))
	for _, b := range enemy_buildings {
		building_zones[b.Location] = true
	}
	soldiers := make([]UnitView, 0)
	reachable := make([]UnitView, 0, len(enemies))
	for _, e := range enemies {
		if building_zones[e.Location] && e.Kind == UnitMilitary {
			continue
		}
		reachable = append(reachable, e)
		if e.Kind == UnitMilitary {
			soldiers = append(soldiers, e)
		}
	}
	local_targets := spreadTargets(members, soldiers, 2)
	production := make([]BuildingView, 0)
	for _, b := range enemy_buildings {
		if b.BuildingID == 1 || b.BuildingID == 22 || b.BuildingID == 23 {
			production = append(production, b)
		}
	}
	for _, u := range members {
		if axialDistance(u.Location.Space, anchor.Location.Space) > 2 {
			orders = append(orders, view.MoveOrder(u, anchor.Location, true))
			continue
		}
		if t, ok := local_targets[u.InternalID]; ok && axialDistance(u.Location.Space, t.Location.Space) <= 0 {
			orders = append(orders, view.AttackUnitOrder(u, t, true))
		} else if b, ok := nearestBuilding(view, u.Location, production); ok {
			orders = append(orders, view.AttackBuildingOrder(u, b, true))
		} else if t, ok := nearestUnit(view, u.Location, reachable); ok && axialDistance(u.Location.Space, t.Location.Space) <= 3 {
			orders = append(orders, view.AttackUnitOrder(u, t, true))
		} else if b, ok := nearestBuilding(view, u.Location, enemy_buildings); ok {
			orders = append(orders, view.AttackBuildingOrder(u, b, true))
		} else if t, ok := nearestUnit(view, u.Location, reachable); ok {
			orders = append(orders, view.AttackUnitOrder(u, t, true))
		} else if st.focus != nil && u.Location.Space != st.focus.Space {
			orders = append(orders, view.MoveOrder(u, *st.focus, true))
		} else {
			st.focus = nil
			if candidates, ok := view.NearestUnexplored(u.Location); ok {
				if target, ok := nearestZone(view, anchor.Location, candidates); ok {
					orders = append(orders, view.MoveOrder(u, target, true))
				}
			}
		}
	}
	return orders
}

// spreadTargets gives each unit its nearest enemy that still has fewer than `per` attackers, so a big group
// doesn't overkill one target; once every enemy is covered the cap rises and assignment continues.
func spreadTargets(units []UnitView, targets []UnitView, per int) map[uint64]UnitView {
	assigned := make(map[uint64]UnitView, len(units))
	if len(targets) == 0 {
		return assigned
	}
	load := make(map[uint64]int, len(targets))
	cap := max(1, per)
	for _, u := range units {
		best, best_distance, found := UnitView{}, -1, false
		for !found {
			for _, t := range targets {
				if load[t.InternalID] >= cap {
					continue
				}
				d := locationDistance(u.Location, t.Location)
				if !found || d < best_distance || (d == best_distance && t.InternalID < best.InternalID) {
					best, best_distance, found = t, d, true
				}
			}
			if !found {
				cap++
			}
		}
		load[best.InternalID]++
		assigned[u.InternalID] = best
	}
	return assigned
}
