package ai

import "sort"

// The Internals fields a spread assignment overwrites while scoring, put back when it finishes
type scoringContext struct {
	unit_id                                                                                               float64
	target, target_from, unit                                                                             *ZoneRef
	target_health, target_max_health, unit_health, unit_stamina, target_assigned, assign_round, target_id float64
}

func saveScoringContext(i *Internals) scoringContext {
	return scoringContext{i.unit_id, i.target, i.target_from, i.unit, i.target_health, i.target_max_health, i.unit_health, i.unit_stamina, i.target_assigned, i.assign_round, i.target_id}
}

func (c scoringContext) restore(i *Internals) {
	i.unit_id = c.unit_id
	i.target, i.target_from, i.unit, i.target_health, i.target_max_health = c.target, c.target_from, c.unit, c.target_health, c.target_max_health
	i.unit_health, i.unit_stamina, i.target_assigned, i.assign_round, i.target_id = c.unit_health, c.unit_stamina, c.target_assigned, c.assign_round, c.target_id
}

// One candidate while units are being shared out over it
type spreadSlot struct {
	c        candidate
	assigned float64
	capacity float64
	capped   bool
}

func (s *spreadSlot) full() bool {
	return s.capped && s.assigned >= s.capacity
}

// Internal id for units and buildings, 0 for anything else, and how many of our units outside exclude already attack it
func (c candidate) idAndAssigned(view View, exclude map[uint64]bool) (float64, int) {
	switch {
	case c.unit != nil:
		return float64(c.unit.InternalID), view.AssignedTo(false, c.unit.InternalID, exclude)
	case c.building != nil:
		return float64(c.building.InternalID), view.AssignedTo(true, c.building.InternalID, exclude)
	}
	return 0, 0
}

func setSpreadUnit(internals *Internals, u *UnitView, loc *ZoneRef, round int) {
	internals.unit_id = float64(u.UnitID)
	internals.unit, internals.unit_health, internals.unit_stamina = loc, u.Health, float64(u.CurrentStamina)
	internals.assign_round = float64(round)
}

// Units in the order they pick: highest unit_order first, ties by internal id
func (p *targetPicker) spreadUnitOrder(view View, internals *Internals, units []UnitView) []UnitView {
	ordered := append([]UnitView(nil), units...)
	keys := make(map[uint64]float64, len(ordered))
	internals.target = nil
	for i := range ordered {
		loc := ordered[i].Location
		setSpreadUnit(internals, &ordered[i], &loc, 0)
		keys[ordered[i].InternalID] = p.spread.unit_order.float(view, internals)
	}
	sort.SliceStable(ordered, func(a, b int) bool {
		ka, kb := keys[ordered[a].InternalID], keys[ordered[b].InternalID]
		return ka > kb || (ka == kb && ordered[a].InternalID < ordered[b].InternalID)
	})
	return ordered
}

func (p *targetPicker) spreadSlots(view View, internals *Internals, center ZoneRef, reordered map[uint64]bool) []spreadSlot {
	slots := make([]spreadSlot, 0)
	candidates := p.candidates(view, internals, center)
	if p.spread.by_space {
		sort.SliceStable(candidates, func(a, b int) bool { return zoneRefLess(candidates[a].loc, candidates[b].loc) })
	}
	seen := make(map[Coordinate]bool)
	for _, c := range candidates {
		id, assigned := c.idAndAssigned(view, reordered)
		if p.spread.by_space {
			if seen[c.loc.Space] {
				continue
			}
			seen[c.loc.Space], assigned = true, 0
		}
		loc := c.loc
		internals.target, internals.target_from, internals.target_id, internals.target_assigned = &loc, &center, id, float64(assigned)
		internals.target_health, internals.target_max_health = c.health()
		slot := spreadSlot{c: c, assigned: float64(assigned)}
		if p.spread.capacity != nil {
			slot.capacity, slot.capped = p.spread.capacity.float(view, internals), true
		}
		slots = append(slots, slot)
	}
	return slots
}

// Scores one unit against every candidate it hasn't taken: the best one with room, else (first round only) the least crowded
func (p *targetPicker) spreadPick(view View, internals *Internals, u UnitView, slots []spreadSlot, taken map[int]bool, round int) (int, bool) {
	loc := u.Location
	setSpreadUnit(internals, &u, &loc, round)
	min_score := p.min_score.float(view, internals)
	best, crowded := -1, -1
	var best_score, crowded_score float64
	for i := range slots {
		if taken[i] {
			continue
		}
		slot := &slots[i]
		at := slot.c.loc
		internals.target, internals.target_from = &at, &loc
		internals.target_health, internals.target_max_health = slot.c.health()
		internals.target_id, _ = slot.c.idAndAssigned(view, nil)
		internals.target_assigned = float64(slot.assigned)
		score := p.score.float(view, internals) + p.spread.unit_score.float(view, internals)
		if p.has_min && score < min_score {
			continue
		}
		if slot.full() {
			if crowded < 0 || slot.assigned < slots[crowded].assigned || (slot.assigned == slots[crowded].assigned && score > crowded_score) {
				crowded, crowded_score = i, score
			}
			continue
		}
		if best < 0 || score > best_score || (score == best_score && zoneRefLess(at, slots[best].c.loc)) {
			best, best_score = i, score
		}
	}
	if best < 0 && round == 0 && p.spread.overflow == spreadOverflowRoundRobin {
		best = crowded
	}
	return best, best >= 0
}

// Shares units out over the picker's candidates in rounds: each round every unit takes its best open candidate (first round clears
// its orders, later rounds queue behind), and a candidate's assigned count includes every earlier pick, so queued picks spread too
func (p *targetPicker) assign(view View, internals *Internals, units []UnitView, act func(UnitView, candidate, bool) Order) []Order {
	center, ok := groupCenter(view, units)
	if !ok {
		return nil
	}
	free, orders := units, make([]Order, 0, len(units))
	if p.together {
		free, orders = regroupFar(view, units, center)
	}
	defer saveScoringContext(internals).restore(internals)
	reordered := make(map[uint64]bool, len(units))
	for _, u := range units {
		reordered[u.InternalID] = true
	}
	slots := p.spreadSlots(view, internals, center, reordered)
	rounds := 1 + max(0, p.spread.queue.int(view, internals))
	ordered := p.spreadUnitOrder(view, internals, free)
	taken := make(map[uint64]map[int]bool, len(ordered))
	for round := 0; round < rounds; round++ {
		for _, u := range ordered {
			i, ok := p.spreadPick(view, internals, u, slots, taken[u.InternalID], round)
			if !ok {
				continue
			}
			first := len(taken[u.InternalID]) == 0
			if first {
				taken[u.InternalID] = map[int]bool{}
			}
			taken[u.InternalID][i] = true
			internals.target_health, internals.target_max_health = slots[i].c.health()
			slots[i].assigned += max(0, p.spread.weight.float(view, internals))
			orders = append(orders, act(u, slots[i].c, first))
		}
	}
	return orders
}

// When a picker works as a group, units too far from its center regroup there instead of picking
func regroupFar(view View, units []UnitView, center ZoneRef) ([]UnitView, []Order) {
	near, orders := make([]UnitView, 0, len(units)), make([]Order, 0)
	for _, u := range units {
		if view.SpaceDistance(u.Location.Space, center.Space) > 2 {
			orders = append(orders, view.MoveOrder(u, center, true))
		} else {
			near = append(near, u)
		}
	}
	return near, orders
}
