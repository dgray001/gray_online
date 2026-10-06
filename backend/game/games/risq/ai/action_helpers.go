package ai

import (
	"github.com/dgray001/gray_online/util"
	"slices"
	"sort"
)

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

func (v *bucketView) ScopedUnits() []UnitView {
	return v.filter(v.View.ScopedUnits())
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

func (v *unbucketedView) ScopedUnits() []UnitView {
	return v.filter(v.View.ScopedUnits())
}

func (v *unbucketedView) IdleUnits() []UnitView {
	return v.filter(v.View.IdleUnits())
}

func (v *unbucketedView) EligibleUnits(kinds ...OrderKind) []UnitView {
	return v.filter(v.View.EligibleUnits(kinds...))
}

type unitWhenView struct {
	View
	internals *Internals
	when      Condition
}

func (v *unitWhenView) filter(units []UnitView) []UnitView {
	saved := v.internals.unit
	defer func() { v.internals.unit = saved }()
	out := make([]UnitView, 0, len(units))
	for _, u := range units {
		loc := u.Location
		v.internals.unit = &loc
		if v.when.Evaluate(v.View, v.internals) {
			out = append(out, u)
		}
	}
	return out
}

func (v *unitWhenView) ScopedUnits() []UnitView {
	return v.filter(v.View.ScopedUnits())
}

func (v *unitWhenView) IdleUnits() []UnitView {
	return v.filter(v.View.IdleUnits())
}

func (v *unitWhenView) EligibleUnits(kinds ...OrderKind) []UnitView {
	return v.filter(v.View.EligibleUnits(kinds...))
}

func unidentifiedUnitCount(view View, space SpaceInfo) int {
	if space.UnitCount == nil {
		return 0
	}
	count := *space.UnitCount
	for _, unit := range view.Units() {
		if unit.GarrisonedIn == nil && unit.Location.Space == space.Space {
			count--
		}
	}
	return max(0, count)
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
	return homeSpaceByUnits(view)
}

// With no buildings: the owned space holding the most of my units, else the space holding the most; ties go to the lower space key
func homeSpaceByUnits(view View) (ZoneRef, bool) {
	counts := map[Coordinate]int{}
	for _, u := range view.Units() {
		counts[u.Location.Space]++
	}
	owned := map[Coordinate]bool{}
	view.CountSpaces(SpaceCondition{Owner: "own"}, func(space Coordinate) bool {
		owned[space] = true
		return false
	})
	var best Coordinate
	found := false
	for space, count := range counts {
		if !found || (owned[space] != owned[best] && owned[space]) || (owned[space] == owned[best] &&
			(count > counts[best] || (count == counts[best] && util.Pair(space.X, space.Y) < util.Pair(best.X, best.Y)))) {
			best, found = space, true
		}
	}
	return ZoneRef{Space: best}, found
}

func baseSpaces(view View) []Coordinate {
	buildings := view.Buildings()
	if len(buildings) == 0 {
		return nil
	}
	occ, ctr, anyVC := map[Coordinate]bool{}, map[Coordinate]bool{}, false
	for _, b := range buildings {
		occ[b.Location.Space] = true
		if b.BuildingID == 1 {
			ctr[b.Location.Space], anyVC = true, true
		}
	}
	nbrs := []Coordinate{{1, 0}, {0, 1}, {-1, 1}, {-1, 0}, {0, -1}, {1, -1}}
	var res []Coordinate
	for s := range occ {
		conn, q, hasC := []Coordinate{s}, []Coordinate{s}, ctr[s]
		delete(occ, s)
		for len(q) > 0 {
			cur := q[0]
			q = q[1:]
			for _, d := range nbrs {
				n := Coordinate{X: cur.X + d.X, Y: cur.Y + d.Y}
				if occ[n] {
					delete(occ, n)
					conn = append(conn, n)
					q = append(q, n)
					if ctr[n] {
						hasC = true
					}
				}
			}
		}
		if !anyVC || hasC {
			res = append(res, conn...)
		}
	}
	return res
}

func zoneRefLess(a ZoneRef, b ZoneRef) bool {
	if a.Space.X != b.Space.X {
		return a.Space.X < b.Space.X
	}
	if a.Space.Y != b.Space.Y {
		return a.Space.Y < b.Space.Y
	}
	if a.Zone.X != b.Zone.X {
		return a.Zone.X < b.Zone.X
	}
	return a.Zone.Y < b.Zone.Y
}

// Picks the candidate closest to from, breaking ties deterministically (lowest ZoneRef).
func nearestZone(view View, from ZoneRef, candidates []ZoneRef) (ZoneRef, bool) {
	best, best_distance, found := ZoneRef{}, -1, false
	for _, c := range candidates {
		d := view.LocationDistance(from, c)
		if !found || d < best_distance || (d == best_distance && zoneRefLess(c, best)) {
			best, best_distance, found = c, d, true
		}
	}
	return best, found
}

func nearestUnexploredOutside(view View, from ZoneRef, min_distance int) ([]ZoneRef, bool) {
	if min_distance <= 0 {
		return view.NearestUnexplored(from)
	}
	best_distance := -1
	var candidates []ZoneRef
	for _, space := range view.AllSpaces() {
		if space.Vision != 0 {
			continue
		}
		distance := view.SpaceDistance(from.Space, space.Space)
		if distance < min_distance {
			continue
		}
		candidate := ZoneRef{Space: space.Space}
		if best_distance < 0 || distance < best_distance {
			candidates, best_distance = []ZoneRef{candidate}, distance
		} else if distance == best_distance {
			candidates = append(candidates, candidate)
		}
	}
	if len(candidates) > 0 {
		return candidates, true
	}
	return view.NearestUnexplored(from)
}

func randomNearestSpace(view View, internals *Internals, from Coordinate, candidates []ZoneRef) (ZoneRef, bool) {
	var tied []ZoneRef
	best_distance := -1
	for _, candidate := range candidates {
		distance := view.SpaceDistance(from, candidate.Space)
		if best_distance < 0 || distance < best_distance {
			tied, best_distance = []ZoneRef{candidate}, distance
		} else if distance == best_distance {
			tied = append(tied, candidate)
		}
	}
	if len(tied) == 0 {
		return ZoneRef{}, false
	}
	index := 0
	if len(tied) > 1 {
		index = internals.rng.Intn(len(tied))
	}
	return tied[index], true
}

func selectFromQueue(view View, internals *Internals, threshold float64, prioritize bool, depth int, kinds ...QKind) []Q {
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
	affordable := make([]Q, 0, len(candidates))
	for _, q := range candidates {
		if canAfford(view, internals, q.Cost) {
			affordable = append(affordable, q)
			if depth > 0 && len(affordable) >= depth {
				break
			}
		}
	}
	return affordable
}

// Unmet cost of every q entry above threshold; weight only selects entries, since scaling costs by it
// would never let a stockpile satisfy the demand
func gatherDemandWeights(view View, internals *Internals, threshold float64) map[ResourceCategory]float64 {
	weights := map[ResourceCategory]float64{ResourceFood: 0, ResourceWood: 0, ResourceStone: 0}
	for _, q := range internals.q {
		if q.Weight <= threshold {
			continue
		}
		weights[ResourceFood] += q.Cost.Food
		weights[ResourceWood] += q.Cost.Wood
		weights[ResourceStone] += q.Cost.Stone
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

func neediestGatherCategory(view View, gatherer UnitView, targets map[ResourceCategory]float64, counts map[ResourceCategory]int) (ResourceCategory, ResourceView, bool) {
	categories := []ResourceCategory{ResourceFood, ResourceWood, ResourceStone}
	deficit := func(c ResourceCategory) float64 { return targets[c] - float64(counts[c]) }
	sort.SliceStable(categories, func(i, j int) bool { return deficit(categories[i]) > deficit(categories[j]) })
	for _, c := range categories {
		if target, ok := view.NearestResource(gatherer.Location, c, gatherer.InternalID); ok {
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
	best, best_distance, found := UnitView{}, -1, false
	for _, u := range units {
		d := view.LocationDistance(from, u.Location)
		if !found || d < best_distance || (d == best_distance && u.InternalID < best.InternalID) {
			best, best_distance, found = u, d, true
		}
	}
	return best, found
}

func nearestBuilding(view View, from ZoneRef, buildings []BuildingView) (BuildingView, bool) {
	best, best_distance, found := BuildingView{}, -1, false
	for _, b := range buildings {
		d := view.LocationDistance(from, b.Location)
		if !found || d < best_distance || (d == best_distance && b.InternalID < best.InternalID) {
			best, best_distance, found = b, d, true
		}
	}
	return best, found
}

func coordinateLess(a Coordinate, b Coordinate) bool {
	if a.X != b.X {
		return a.X < b.X
	}
	return a.Y < b.Y
}

func nearestEnemySpace(view View, from ZoneRef, buildings []BuildingView) (Coordinate, bool) {
	seen := make(map[Coordinate]bool)
	best, best_distance, found := Coordinate{}, -1, false
	for _, b := range buildings {
		space := b.Location.Space
		if seen[space] {
			continue
		}
		seen[space] = true
		d := view.SpaceDistance(from.Space, space)
		if !found || d < best_distance || (d == best_distance && coordinateLess(space, best)) {
			best, best_distance, found = space, d, true
		}
	}
	return best, found
}

func nearestEnemyZone(view View, from ZoneRef, units []UnitView, buildings []BuildingView) (ZoneRef, bool) {
	best, best_distance, found := ZoneRef{}, -1, false
	consider := func(loc ZoneRef) {
		d := view.LocationDistance(from, loc)
		if !found || d < best_distance || (d == best_distance && zoneRefLess(loc, best)) {
			best, best_distance, found = loc, d, true
		}
	}
	for _, u := range units {
		consider(u.Location)
	}
	for _, b := range buildings {
		consider(b.Location)
	}
	return best, found
}

// queue > 1 tops each building up to that many production orders, so a unit finishing mid-turn
// hands its leftover stamina to the next one instead of losing it
func createUnits(view View, internals *Internals, unit_id uint32, buildings buildingFilter, queue int) []Order {
	candidates := view.IdleBuildings()
	if queue > 1 {
		candidates = view.Buildings()
	}
	orders := make([]Order, 0)
	for _, b := range candidates {
		if !buildings.matches(b.BuildingID) || b.UnderConstruction {
			continue
		}
		for queued := productionOrderCount(b); queued < max(1, queue); queued++ {
			if current, limit := internals.population(view); current >= limit {
				if util.DebugLog.Writer() != nil {
					util.DebugLog.Printf("ai %s: REJECTED create unit_id %d: housing capped (%d / %d)", view.Nickname(), unit_id, current, limit)
				}
				return orders
			}
			p, ok := unitProducible(b, unit_id)
			if !ok {
				break
			}
			if !canAfford(view, internals, p.Cost) {
				if util.DebugLog.Writer() != nil {
					util.DebugLog.Printf("ai %s: REJECTED create unit_id %d: insufficient resources (cost: %+v)", view.Nickname(), unit_id, p.Cost)
				}
				break
			}
			if util.DebugLog.Writer() != nil {
				util.DebugLog.Printf("ai %s: SUCCESS create unit_id %d (cost: %+v)", view.Nickname(), unit_id, p.Cost)
			}
			orders = append(orders, view.CreateUnitOrder(b, p.ID))
			internals.spend(p.Cost)
			internals.pending_population++
		}
	}
	return orders
}

func unitProducible(b BuildingView, unit_id uint32) (Producible, bool) {
	for _, p := range b.Producibles {
		if p.Kind == ProducibleUnit && p.ID == unit_id {
			return p, true
		}
	}
	return Producible{}, false
}

func productionOrderCount(b BuildingView) int {
	count := b.PlannedProduction
	for _, o := range b.ActiveOrders {
		if o.Kind == BuildingOrderCreate || o.Kind == BuildingOrderResearch {
			count++
		}
	}
	return count
}

func researchTech(view View, internals *Internals, tech_id uint32, buildings buildingFilter, queue int) []Order {
	candidates := view.IdleBuildings()
	if queue > 1 {
		candidates = view.Buildings()
	}
	for _, b := range candidates {
		if !buildings.matches(b.BuildingID) || b.UnderConstruction || productionOrderCount(b) >= max(1, queue) {
			continue
		}
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

// With no ids or types set every unit matches
func (f unitFilter) matches(u UnitView) bool {
	return len(f.unit_ids) == 0 && len(f.unit_types) == 0 || f.unit_ids[u.UnitID] || f.unit_types[u.Type]
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

func canBuild(u UnitView, building_id uint32) bool {
	return slices.ContainsFunc(u.Builds, func(p Producible) bool { return p.ID == building_id })
}

func buildWith(view View, internals *Internals, units []UnitView, building_id uint32, max_builders int, site *spaceQuery, picker *sitePicker, reuse bool) []Order {
	units = slices.DeleteFunc(slices.Clone(units), func(u UnitView) bool { return !canBuild(u, building_id) })
	if len(units) == 0 || !view.BuildingAvailable(building_id) {
		return nil
	}
	var target ZoneRef
	ok := false
	if reuse {
		target, ok = unbuiltFoundation(view, units[0].Location, building_id)
	}
	if !ok {
		cost := view.BuildCost(building_id)
		if !canAfford(view, internals, cost) {
			if util.DebugLog.Writer() != nil {
				util.DebugLog.Printf("ai %s: REJECTED build building_id %d: insufficient resources (cost: %+v)", view.Nickname(), building_id, cost)
			}
			return nil
		}
		from := units[0].Location
		var include func(Coordinate) bool
		if site != nil {
			reference, site_include, valid := site.buildFilter(view, internals)
			if !valid {
				return nil
			}
			include = site_include
			if picker == nil || site.has_from {
				from = reference
			}
		}
		if picker != nil {
			target, ok = picker.best(view, internals, from, building_id, include)
		} else {
			target, ok = view.NearestBuildSite(from, building_id, include)
		}
		if !ok {
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

// Scores every site a building could go on with an expression and picks the best; none when no site reaches min_score
type sitePicker struct {
	score     amount
	min_score amount
	has_min   bool
}

func (p *sitePicker) best(view View, internals *Internals, from ZoneRef, building_id uint32, include func(Coordinate) bool) (ZoneRef, bool) {
	saved_target, saved_from := internals.target, internals.target_from
	defer func() { internals.target, internals.target_from = saved_target, saved_from }()
	min_score := p.min_score.float(view, internals)
	var best ZoneRef
	best_score, found := 0.0, false
	for _, site := range view.BuildSites(building_id, include) {
		candidate := site
		internals.target, internals.target_from = &candidate, &from
		score := p.score.float(view, internals)
		if p.has_min && score < min_score {
			continue
		}
		if !found || score > best_score || (score == best_score && closerSite(view, from, site, best)) {
			best, best_score, found = site, score, true
		}
	}
	return best, found
}

// Ties go to the site nearer the builders, then the lowest zone ref
func closerSite(view View, from ZoneRef, a ZoneRef, b ZoneRef) bool {
	da, db := view.LocationDistance(from, a), view.LocationDistance(from, b)
	return da < db || (da == db && zoneRefLess(a, b))
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
