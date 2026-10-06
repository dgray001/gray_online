package ai

// armyAction runs the whole military as two groups: defenders (everything not in the assault) hold home and
// converge on threats near any of our buildings, while an assault group of the chosen unit types launches once
// enough of them are gathered at home, stays together, and falls back when too few survive.
type armyAction struct {
	assault_ids   map[uint32]bool
	launch        amount
	retreat       amount
	defend_radius amount
	// launches an all-units strike right after a big enemy army was broken, if we still have this many (0 disables)
	strike amount
}

type armyState struct {
	assault bool
	members map[uint64]bool
}

func (a *armyAction) inAssaultGroup(u UnitView) bool {
	return len(a.assault_ids) == 0 || a.assault_ids[u.UnitID]
}

func (a *armyAction) ToOrders(view View, internals *Internals) []Order {
	launch, retreat := a.launch.int(view, internals), a.retreat.int(view, internals)
	defend_radius := a.defend_radius.int(view, internals)
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

	if !st.assault {
		gathered := make([]UnitView, 0)
		for _, u := range mil {
			if a.inAssaultGroup(u) && view.SpaceDistance(u.Location.Space, home.Space) <= 2 {
				gathered = append(gathered, u)
			}
		}
		if len(gathered) >= launch {
			st.assault = true
			st.members = make(map[uint64]bool, len(gathered))
			for _, u := range gathered {
				st.members[u.InternalID] = true
			}
		}
	} else if len(st.members) <= retreat {
		st.assault = false
		st.members = make(map[uint64]bool)
	}

	enemies := view.VisibleEnemyUnits()
	threats := make([]UnitView, 0)
	for _, e := range enemies {
		if view.SpaceDistance(home.Space, e.Location.Space) <= defend_radius {
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
	defender_targets := spreadTargets(view, defenders, threats)
	for _, u := range defenders {
		if t, ok := defender_targets[u.InternalID]; ok {
			orders = append(orders, view.AttackUnitOrder(u, t, true))
		} else if len(threats) == 0 && u.CurrentOrder == nil && view.SpaceDistance(u.Location.Space, home.Space) >= 1 {
			orders = append(orders, view.MoveOrder(u, home, true))
		}
	}
	if len(members) == 0 {
		return orders
	}
	if !st.assault {
		return orders
	}
	return append(orders, a.assaultOrders(view, members, enemies)...)
}

func (a *armyAction) assaultOrders(view View, members []UnitView, enemies []UnitView) []Order {
	orders := make([]Order, 0, len(members))
	enemy_buildings := view.VisibleEnemyBuildings()
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
	local_targets := spreadTargets(view, members, soldiers)
	for _, u := range members {
		if t, ok := local_targets[u.InternalID]; ok && view.SpaceDistance(u.Location.Space, t.Location.Space) <= 0 {
			orders = append(orders, view.AttackUnitOrder(u, t, true))
		} else if t, ok := nearestUnit(view, u.Location, reachable); ok {
			orders = append(orders, view.AttackUnitOrder(u, t, true))
		} else if b, ok := nearestBuilding(view, u.Location, enemy_buildings); ok {
			orders = append(orders, view.AttackBuildingOrder(u, b, true))
		} else if candidates, ok := view.NearestUnexplored(u.Location); ok {
			if target, ok := nearestZone(view, u.Location, candidates); ok {
				orders = append(orders, view.MoveOrder(u, target, true))
			}
		}
	}
	return orders
}

// spreadTargets gives each unit its nearest enemy that still has fewer than `per` attackers, so a big group
// doesn't overkill one target; once every enemy is covered the cap rises and assignment continues.
func spreadTargets(view View, units []UnitView, targets []UnitView) map[uint64]UnitView {
	assigned := make(map[uint64]UnitView, len(units))
	if len(targets) == 0 {
		return assigned
	}
	load := make(map[uint64]int, len(targets))
	for _, u := range units {
		best, best_dist, best_load := targets[0], view.LocationDistance(u.Location, targets[0].Location), load[targets[0].InternalID]
		for _, t := range targets[1:] {
			l, d := load[t.InternalID], view.LocationDistance(u.Location, t.Location)
			if l < best_load || (l == best_load && (d < best_dist || (d == best_dist && t.InternalID < best.InternalID))) {
				best, best_dist, best_load = t, d, l
			}
		}
		load[best.InternalID]++
		assigned[u.InternalID] = best
	}
	return assigned
}
