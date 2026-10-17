package ai

import (
	"slices"
	"sort"

	"github.com/dgray001/gray_online/util"
)

type addQAction struct {
	q_type QKind
	id     *uint32
	cost   Cost
	weight amount
}

func (a *addQAction) ToOrders(view View, internals *Internals) []Order {
	q := Q{Type: a.q_type, Weight: a.weight.float(view, internals)}
	switch a.q_type {
	case QResource:
		q.Cost = a.cost
	case QUnit:
		q.ID = a.id
		q.Cost = view.UnitCost(*a.id)
	case QBuilding:
		q.ID = a.id
		q.Cost = view.BuildCost(*a.id)
	case QTech:
		q.ID = a.id
		q.Cost = view.TechCost(*a.id)
	}
	internals.q = append(internals.q, q)
	return nil
}

type buildNextInQAction struct {
	filtered
	eligible   []OrderKind
	weight     amount
	prioritize bool
	depth      amount
	max        amount
}

type createNextInQAction struct {
	weight     amount
	prioritize bool
	depth      amount
}

type researchNextInQAction struct {
	weight     amount
	prioritize bool
	depth      amount
}

type produceNextInQAction struct {
	filtered
	weight     amount
	prioritize bool
	depth      amount
	max        amount
}

func (a *buildNextInQAction) ToOrders(view View, internals *Internals) []Order {
	limit := a.max.int(view, internals)
	units := a.filter.apply(eligibleUnits(view, a.eligible), isEconomic)
	for _, q := range selectFromQueue(view, internals, a.weight.float(view, internals), a.prioritize, a.depth.int(view, internals), QBuilding) {
		if orders := buildWith(view, internals, units, *q.ID, limit, nil, nil, true); len(orders) > 0 {
			return orders
		}
	}
	return nil
}

func (a *createNextInQAction) ToOrders(view View, internals *Internals) []Order {
	for _, q := range selectFromQueue(view, internals, a.weight.float(view, internals), a.prioritize, a.depth.int(view, internals), QUnit) {
		if orders := createUnits(view, internals, *q.ID, buildingFilter{}, 1); len(orders) > 0 {
			return orders
		}
	}
	return nil
}

func (a *researchNextInQAction) ToOrders(view View, internals *Internals) []Order {
	for _, q := range selectFromQueue(view, internals, a.weight.float(view, internals), a.prioritize, a.depth.int(view, internals), QTech) {
		if orders := researchTech(view, internals, *q.ID, buildingFilter{}, 1); len(orders) > 0 {
			return orders
		}
	}
	return nil
}

func (a *produceNextInQAction) ToOrders(view View, internals *Internals) []Order {
	limit := a.max.int(view, internals)
	units := a.filter.apply(view.IdleUnits(), isEconomic)
	for _, q := range selectFromQueue(view, internals, a.weight.float(view, internals), a.prioritize, a.depth.int(view, internals), QBuilding, QUnit, QTech) {
		var orders []Order
		switch q.Type {
		case QBuilding:
			orders = buildWith(view, internals, units, *q.ID, limit, nil, nil, true)
		case QUnit:
			orders = createUnits(view, internals, *q.ID, buildingFilter{}, 1)
		case QTech:
			orders = researchTech(view, internals, *q.ID, buildingFilter{}, 1)
		}
		if len(orders) > 0 {
			return orders
		}
	}
	return nil
}

// hireAction buys mercenaries with gold at the home village center once Mercenary Contracts is known.
// Mercenary price mirrors risq.mercenaryCost: (food + wood + stone + 1.5*gold) * 1.3, paid in gold.
type hireAction struct {
	unit_id uint32
	max     amount
	reserve amount
}

func (a *hireAction) ToOrders(view View, internals *Internals) []Order {
	limit, reserve := a.max.int(view, internals), a.reserve.float(view, internals)
	if !view.TechResearched(4) {
		return nil
	}
	home, ok := homeLocation(view)
	if !ok {
		return nil
	}
	cost := view.UnitCost(a.unit_id)
	price := (cost.Food + cost.Wood + cost.Stone + 1.5*cost.Gold) * 1.3
	orders := make([]Order, 0)
	for limit <= 0 || len(orders) < limit {
		current, limit := internals.population(view)
		if current >= limit || internals.available(view, ResourceGold) < price+reserve {
			break
		}
		internals.spend(Cost{Gold: price})
		internals.pending_population++
		orders = append(orders, view.HireMercenaryOrder(a.unit_id, home))
	}
	return orders
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
				util.Debugf(view.Logger(), "ai %s: REJECTED create unit_id %d: housing capped (%d / %d)", view.Nickname(), unit_id, current, limit)
				return orders
			}
			p, ok := unitProducible(b, unit_id)
			if !ok {
				break
			}
			if !canAfford(view, internals, p.Cost) {
				util.Debugf(view.Logger(), "ai %s: REJECTED create unit_id %d: insufficient resources (cost: %+v)", view.Nickname(), unit_id, p.Cost)
				break
			}
			util.Debugf(view.Logger(), "ai %s: SUCCESS create unit_id %d (cost: %+v)", view.Nickname(), unit_id, p.Cost)
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
			util.Debugf(view.Logger(), "ai %s: REJECTED build building_id %d: insufficient resources (cost: %+v)", view.Nickname(), building_id, cost)
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
