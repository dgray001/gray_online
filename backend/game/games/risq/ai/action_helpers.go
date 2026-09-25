package ai

import "sort"

func canAfford(view View, internals *Internals, cost Cost) bool {
	for _, c := range []ResourceCategory{ResourceFood, ResourceWood, ResourceStone, ResourceGold} {
		if internals.available(view, c) < cost.of(c) {
			return false
		}
	}
	return true
}

func eligibleUnits(view View, eligible []OrderKind) []UnitView {
	if len(eligible) == 0 {
		return view.IdleUnits()
	}
	return view.EligibleUnits(eligible...)
}

type bucketView struct {
	View
	members map[uint64]bool
}

func (v *bucketView) filter(units []UnitView) []UnitView {
	out := make([]UnitView, 0, len(units))
	for _, u := range units {
		if v.members[u.InternalID] {
			out = append(out, u)
		}
	}
	return out
}

func (v *bucketView) IdleUnits() []UnitView {
	return v.filter(v.View.IdleUnits())
}

func (v *bucketView) EligibleUnits(kinds ...OrderKind) []UnitView {
	return v.filter(v.View.EligibleUnits(kinds...))
}

type unbucketedView struct {
	View
	internals *Internals
}

func (v *unbucketedView) filter(units []UnitView) []UnitView {
	out := make([]UnitView, 0, len(units))
	for _, u := range units {
		if !v.internals.isBucketed(u.InternalID) {
			out = append(out, u)
		}
	}
	return out
}

func (v *unbucketedView) IdleUnits() []UnitView {
	return v.filter(v.View.IdleUnits())
}

func (v *unbucketedView) EligibleUnits(kinds ...OrderKind) []UnitView {
	return v.filter(v.View.EligibleUnits(kinds...))
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
		if canAfford(view, internals, q.Cost) {
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
		weights[c] -= internals.available(view, c)
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
		if u.CurrentOrder != nil && u.CurrentOrder.TargetResource != nil {
			counts[u.CurrentOrder.TargetResource.Category]++
		}
	}
	return counts
}

func gatherTargets(demand map[ResourceCategory]float64, counts map[ResourceCategory]int, new_workers int) map[ResourceCategory]float64 {
	total_demand := demand[ResourceFood] + demand[ResourceWood] + demand[ResourceStone]
	total_workers := float64(counts[ResourceFood] + counts[ResourceWood] + counts[ResourceStone] + new_workers)
	targets := make(map[ResourceCategory]float64, 3)
	for c, d := range demand {
		targets[c] = d / total_demand * total_workers
	}
	return targets
}

func neediestGatherCategory(view View, from ZoneRef, targets map[ResourceCategory]float64, counts map[ResourceCategory]int) (ResourceCategory, ResourceView, bool) {
	categories := []ResourceCategory{ResourceFood, ResourceWood, ResourceStone}
	deficit := func(c ResourceCategory) float64 { return targets[c] - float64(counts[c]) }
	sort.SliceStable(categories, func(i, j int) bool { return deficit(categories[i]) > deficit(categories[j]) })
	for _, c := range categories {
		if target, ok := view.NearestResource(from, c); ok {
			return c, target, true
		}
	}
	return 0, ResourceView{}, false
}

func pickUnitTarget(view View, target attackTarget, from ZoneRef, units []UnitView) (UnitView, bool) {
	var preferred UnitKind
	switch target {
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

func createUnits(view View, internals *Internals, unit_id uint32) []Order {
	orders := make([]Order, 0)
	for _, b := range view.IdleBuildings() {
		if current, limit := internals.population(view); current >= limit {
			break
		}
		for _, p := range b.Producibles {
			if p.Kind == ProducibleUnit && p.ID == unit_id && canAfford(view, internals, p.Cost) {
				orders = append(orders, view.CreateUnitOrder(b, p.ID))
				internals.spend(p.Cost)
				internals.pending_population++
				break
			}
		}
	}
	return orders
}

func researchTech(view View, internals *Internals, tech_id uint32) []Order {
	for _, b := range view.IdleBuildings() {
		for _, p := range b.Producibles {
			if p.Kind == ProducibleTech && p.ID == tech_id && canAfford(view, internals, p.Cost) {
				internals.spend(p.Cost)
				return []Order{view.ResearchOrder(b, p.ID)}
			}
		}
	}
	return nil
}

type unitFilter struct {
	unit_ids   map[uint32]bool
	unit_types map[UnitType]bool
}

func (f unitFilter) apply(units []UnitView, fallback func(UnitView) bool) []UnitView {
	out := make([]UnitView, 0, len(units))
	for _, u := range units {
		if (len(f.unit_ids) == 0 && len(f.unit_types) == 0 && fallback(u)) || f.unit_ids[u.UnitID] || f.unit_types[u.Type] {
			out = append(out, u)
		}
	}
	return out
}

func isEconomic(u UnitView) bool { return u.Kind == UnitEconomic }
func isMilitary(u UnitView) bool { return u.Kind == UnitMilitary }
func anyUnit(UnitView) bool      { return true }

type filtered struct {
	filter unitFilter
}

func (f *filtered) setFilter(filter unitFilter) {
	f.filter = filter
}

func buildWith(view View, internals *Internals, units []UnitView, building_id uint32, max_builders int) []Order {
	if len(units) == 0 {
		return nil
	}
	target, ok := unbuiltFoundation(view, units[0].Location, building_id)
	if !ok {
		cost := view.BuildCost(building_id)
		if !canAfford(view, internals, cost) {
			return nil
		}
		if target, ok = view.NearestBuildSite(units[0].Location, building_id); !ok {
			return nil
		}
		internals.spend(cost)
	}
	orders := make([]Order, 0)
	for _, u := range units {
		if len(orders) >= max(1, max_builders) {
			break
		}
		orders = append(orders, view.BuildOrder(u, building_id, target, true))
	}
	return orders
}

func unbuiltFoundation(view View, from ZoneRef, building_id uint32) (ZoneRef, bool) {
	candidates := make([]ZoneRef, 0)
	for _, f := range view.Foundations() {
		if f.BuildingID == building_id && f.Builders == 0 {
			candidates = append(candidates, f.Location)
		}
	}
	return nearestZone(view, from, candidates)
}

func withoutUnit(units []UnitView, internal_id uint64) []UnitView {
	out := make([]UnitView, 0, len(units))
	for _, u := range units {
		if u.InternalID != internal_id {
			out = append(out, u)
		}
	}
	return out
}

type buildingFilter struct {
	ids map[uint32]bool
}

func (f buildingFilter) matches(building_id uint32) bool {
	return len(f.ids) == 0 || f.ids[building_id]
}

func (f buildingFilter) apply(buildings []BuildingView) []BuildingView {
	out := make([]BuildingView, 0, len(buildings))
	for _, b := range buildings {
		if f.matches(b.BuildingID) {
			out = append(out, b)
		}
	}
	return out
}

type buildingFiltered struct {
	buildings buildingFilter
}

func (f *buildingFiltered) setBuildingFilter(filter buildingFilter) {
	f.buildings = filter
}

func inAttackRange(view View, b BuildingView, units []UnitView, buildings []BuildingView) ([]UnitView, []BuildingView) {
	units_in_range := make([]UnitView, 0)
	for _, u := range units {
		if view.InAttackRange(b, u.Location) {
			units_in_range = append(units_in_range, u)
		}
	}
	buildings_in_range := make([]BuildingView, 0)
	for _, target := range buildings {
		if view.InAttackRange(b, target.Location) {
			buildings_in_range = append(buildings_in_range, target)
		}
	}
	return units_in_range, buildings_in_range
}

func assignedBuildings(view View, kind OrderKind) map[uint64]bool {
	assigned := make(map[uint64]bool)
	for _, u := range view.Units() {
		if u.CurrentOrder != nil && u.CurrentOrder.Kind == kind && u.CurrentOrder.TargetBuilding != nil {
			assigned[u.CurrentOrder.TargetBuilding.InternalID] = true
		}
	}
	return assigned
}
