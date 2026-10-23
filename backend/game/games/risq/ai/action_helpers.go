package ai

import (
	"slices"
	"sort"

	"github.com/dgray001/gray_online/util"
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

// How many of my units that are not garrisoned stand in each space
func ownUnitsBySpace(view View) map[Coordinate]int {
	counts := map[Coordinate]int{}
	for _, unit := range view.Units() {
		if unit.GarrisonedIn == nil {
			counts[unit.Location.Space]++
		}
	}
	return counts
}

func unidentifiedUnitCount(space SpaceInfo, own map[Coordinate]int) int {
	if space.UnitCount == nil {
		return 0
	}
	return max(0, *space.UnitCount-own[space.Space])
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

func nearestExplorationTargets(view View, internals *Internals, from ZoneRef, min_distance int, include_fog float64) ([]ZoneRef, bool) {
	if min_distance <= 0 && include_fog < 0 {
		return view.NearestUnexplored(from)
	}
	best_distance := -1
	var candidates []ZoneRef
	for _, space := range view.AllSpaces() {
		fog_allowed := space.Vision == 1 && include_fog >= 0 && float64(view.TurnNumber()-internals.last_observed[space.Space]) >= include_fog
		if space.Vision != 0 && !fog_allowed {
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
	if min_distance > 0 {
		return nearestExplorationTargets(view, internals, from, 0, include_fog)
	}
	return nil, false
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

func withoutUnit(units []UnitView, internal_id uint64) []UnitView {
	out := make([]UnitView, 0, len(units))
	for _, u := range units {
		if u.InternalID != internal_id {
			out = append(out, u)
		}
	}
	return out
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
