package ai

import "sort"

func canAfford(view View, cost Cost) bool {
	return view.Resource(ResourceFood) >= cost.Food &&
		view.Resource(ResourceWood) >= cost.Wood &&
		view.Resource(ResourceStone) >= cost.Stone &&
		view.Resource(ResourceGold) >= cost.Gold
}

func eligibleUnits(view View, eligible []OrderKind) []UnitView {
	if len(eligible) == 0 {
		return view.IdleUnits()
	}
	return view.EligibleUnits(eligible...)
}

func homeLocation(view View) (ZoneRef, bool) {
	buildings := view.Buildings()
	for _, b := range buildings {
		if b.BuildingID == 1 {
			return b.Location, true
		}
	}
	if len(buildings) > 0 {
		return buildings[0].Location, true
	}
	return ZoneRef{}, false
}

// Picks the candidate closest to from, randomly breaking ties
func nearestZone(view View, from ZoneRef, candidates []ZoneRef) (ZoneRef, bool) {
	var tied []ZoneRef
	best_distance := -1
	for _, c := range candidates {
		d := locationDistance(from, c)
		if best_distance == -1 || d < best_distance {
			tied, best_distance = []ZoneRef{c}, d
		} else if d == best_distance {
			tied = append(tied, c)
		}
	}
	if len(tied) == 0 {
		return ZoneRef{}, false
	}
	return tied[view.RandomIntn(len(tied))], true
}

// First affordable queued entry above threshold weight: queue order, or highest-weight first if prioritize.
func selectFromQueue(view View, internals *Internals, threshold float64, prioritize bool, kinds ...QKind) (Q, bool) {
	allowed := make(map[QKind]bool, len(kinds))
	for _, k := range kinds {
		allowed[k] = true
	}
	candidates := make([]Q, 0, len(internals.q))
	for _, q := range internals.q {
		if allowed[q.Type] && q.Weight > threshold {
			candidates = append(candidates, q)
		}
	}
	if prioritize {
		sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].Weight > candidates[j].Weight })
	}
	for _, q := range candidates {
		if canAfford(view, q.Cost) {
			return q, true
		}
	}
	return Q{}, false
}

// Demand signal from weighted q minus current resources
func gatherDemandWeights(view View, internals *Internals, threshold float64) map[ResourceCategory]float64 {
	weights := map[ResourceCategory]float64{ResourceFood: 0, ResourceWood: 0, ResourceStone: 0}
	for _, q := range internals.q {
		w := q.Weight - threshold
		if w <= 0 {
			continue
		}
		weights[ResourceFood] += q.Cost.Food * w
		weights[ResourceWood] += q.Cost.Wood * w
		weights[ResourceStone] += q.Cost.Stone * w
	}
	for _, c := range []ResourceCategory{ResourceFood, ResourceWood, ResourceStone} {
		weights[c] -= view.Resource(c)
		if weights[c] < 0 {
			weights[c] = 0
		}
	}
	if weights[ResourceFood]+weights[ResourceWood]+weights[ResourceStone] <= 0 {
		return map[ResourceCategory]float64{ResourceFood: 1, ResourceWood: 1, ResourceStone: 1}
	}
	return weights
}

func currentGatherCounts(view View) map[ResourceCategory]int {
	counts := map[ResourceCategory]int{ResourceFood: 0, ResourceWood: 0, ResourceStone: 0}
	for _, u := range view.Units() {
		if u.CurrentOrder != nil && u.CurrentOrder.GatherCategory != nil {
			counts[*u.CurrentOrder.GatherCategory]++
		}
	}
	return counts
}

// Ranks categories by how far below their demand-weighted target share they are, then picks
func neediestGatherCategory(view View, from ZoneRef, demand map[ResourceCategory]float64, counts map[ResourceCategory]int) (ResourceCategory, ZoneRef, bool) {
	total_demand := demand[ResourceFood] + demand[ResourceWood] + demand[ResourceStone]
	total_workers := counts[ResourceFood] + counts[ResourceWood] + counts[ResourceStone] + 1
	categories := []ResourceCategory{ResourceFood, ResourceWood, ResourceStone}
	deficit := func(c ResourceCategory) float64 {
		return demand[c]/total_demand - float64(counts[c])/float64(total_workers)
	}
	sort.Slice(categories, func(i, j int) bool { return deficit(categories[i]) > deficit(categories[j]) })
	for _, c := range categories {
		if target, ok := view.NearestResource(from, c); ok {
			return c, target, true
		}
	}
	return 0, ZoneRef{}, false
}

func (a *attackAction) pickTarget(view View, from ZoneRef, units []UnitView) (UnitView, bool) {
	var preferred UnitKind
	switch a.target {
	case attackTargetEconomic:
		preferred = UnitEconomic
	case attackTargetAny:
		return nearestUnit(view, from, units)
	default:
		preferred = UnitMilitary
	}
	if target, ok := nearestUnitOfKind(view, from, units, preferred); ok {
		return target, true
	}
	return nearestUnit(view, from, units)
}

func nearestUnitOfKind(view View, from ZoneRef, units []UnitView, kind UnitKind) (UnitView, bool) {
	filtered := make([]UnitView, 0, len(units))
	for _, u := range units {
		if u.Kind == kind {
			filtered = append(filtered, u)
		}
	}
	return nearestUnit(view, from, filtered)
}

func nearestUnit(view View, from ZoneRef, units []UnitView) (UnitView, bool) {
	var tied []UnitView
	best_distance := -1
	for _, u := range units {
		d := locationDistance(from, u.Location)
		if best_distance == -1 || d < best_distance {
			tied, best_distance = []UnitView{u}, d
		} else if d == best_distance {
			tied = append(tied, u)
		}
	}
	if len(tied) == 0 {
		return UnitView{}, false
	}
	return tied[view.RandomIntn(len(tied))], true
}

func nearestBuilding(view View, from ZoneRef, buildings []BuildingView) (BuildingView, bool) {
	var tied []BuildingView
	best_distance := -1
	for _, b := range buildings {
		d := locationDistance(from, b.Location)
		if best_distance == -1 || d < best_distance {
			tied, best_distance = []BuildingView{b}, d
		} else if d == best_distance {
			tied = append(tied, b)
		}
	}
	if len(tied) == 0 {
		return BuildingView{}, false
	}
	return tied[view.RandomIntn(len(tied))], true
}

func nearestEnemySpace(view View, from ZoneRef, buildings []BuildingView) (Coordinate, bool) {
	seen := make(map[Coordinate]bool)
	var tied []Coordinate
	best_distance := -1
	for _, b := range buildings {
		space := b.Location.Space
		if seen[space] {
			continue
		}
		seen[space] = true
		d := axialDistance(from.Space, space)
		if best_distance == -1 || d < best_distance {
			tied, best_distance = []Coordinate{space}, d
		} else if d == best_distance {
			tied = append(tied, space)
		}
	}
	if len(tied) == 0 {
		return Coordinate{}, false
	}
	return tied[view.RandomIntn(len(tied))], true
}

func nearestEnemyZone(view View, from ZoneRef, units []UnitView, buildings []BuildingView) (ZoneRef, bool) {
	var tied []ZoneRef
	best_distance := -1
	consider := func(loc ZoneRef) {
		d := locationDistance(from, loc)
		if best_distance == -1 || d < best_distance {
			tied, best_distance = []ZoneRef{loc}, d
		} else if d == best_distance {
			tied = append(tied, loc)
		}
	}
	for _, u := range units {
		consider(u.Location)
	}
	for _, b := range buildings {
		consider(b.Location)
	}
	if len(tied) == 0 {
		return ZoneRef{}, false
	}
	return tied[view.RandomIntn(len(tied))], true
}
