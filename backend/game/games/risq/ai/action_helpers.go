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

// Sums resource costs of everything this player can currently build/produce/research, as a demand signal
func gatherDemandWeights(view View) map[ResourceCategory]float64 {
	weights := map[ResourceCategory]float64{ResourceFood: 0, ResourceWood: 0, ResourceStone: 0}
	add := func(c Cost) {
		weights[ResourceFood] += c.Food
		weights[ResourceWood] += c.Wood
		weights[ResourceStone] += c.Stone
	}
	for _, b := range view.Buildings() {
		for _, p := range b.Producibles {
			add(p.Cost)
		}
	}
	seen := make(map[uint32]bool)
	for _, u := range view.Units() {
		for _, p := range u.Builds {
			if seen[p.ID] {
				continue
			}
			seen[p.ID] = true
			add(p.Cost)
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
