package ai

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Where a "within" distance is measured from: our home (default), the center of a bucket's units, or the
// candidate currently being scored by a target picker.
type anchorKind uint8

const (
	anchorHome anchorKind = iota
	anchorBucket
	anchorTarget
	// the unit a "unit_when" condition is being checked for
	anchorUnit
)

type anchor struct {
	kind   anchorKind
	bucket string
}

func parseAnchor(raw any) (anchor, error) {
	if raw == nil {
		return anchor{kind: anchorHome}, nil
	}
	name, ok := raw.(string)
	if !ok || name == "" {
		return anchor{}, fmt.Errorf("\"from\" must be \"home\", \"target\", \"unit\" or a bucket name")
	}
	switch name {
	case "home":
		return anchor{kind: anchorHome}, nil
	case "target":
		return anchor{kind: anchorTarget}, nil
	case "unit":
		return anchor{kind: anchorUnit}, nil
	}
	return anchor{kind: anchorBucket, bucket: name}, nil
}

func (a anchor) location(view View, internals *Internals) (ZoneRef, bool) {
	switch a.kind {
	case anchorBucket:
		b := internals.Buckets[a.bucket]
		if b == nil {
			return ZoneRef{}, false
		}
		members := make([]UnitView, 0, len(b.Members))
		for _, u := range view.Units() {
			if b.Members[u.InternalID] {
				members = append(members, u)
			}
		}
		return groupCenter(members)
	case anchorTarget:
		if internals.target == nil {
			return ZoneRef{}, false
		}
		return *internals.target, true
	case anchorUnit:
		if internals.unit == nil {
			return ZoneRef{}, false
		}
		return *internals.unit, true
	}
	return homeLocation(view)
}

// The unit closest to all the others, so a group's center is always a real, reachable zone
func groupCenter(units []UnitView) (ZoneRef, bool) {
	if len(units) == 0 {
		return ZoneRef{}, false
	}
	best, best_sum := units[0].Location, -1
	for _, u := range units {
		sum := 0
		for _, other := range units {
			sum += axialDistance(u.Location.Space, other.Location.Space)
		}
		if best_sum < 0 || sum < best_sum {
			best, best_sum = u.Location, sum
		}
	}
	return best, true
}

// Optional "within" (spaces) around an anchor; without "within" everything counts
type nearFilter struct {
	within     int
	has_within bool
	from       anchor
}

func parseNearFilter(obj map[string]any) (nearFilter, error) {
	f := nearFilter{}
	if w, ok := obj["within"].(float64); ok {
		f.within, f.has_within = int(w), true
	}
	from, err := parseAnchor(obj["from"])
	if err != nil {
		return f, err
	}
	if obj["from"] != nil && !f.has_within {
		return f, fmt.Errorf("\"from\" needs a \"within\" distance")
	}
	f.from = from
	return f, nil
}

func (f nearFilter) contains(view View, internals *Internals, loc ZoneRef) bool {
	if !f.has_within {
		return true
	}
	center, ok := f.from.location(view, internals)
	return ok && axialDistance(center.Space, loc.Space) <= f.within
}

// ---------- target picking for attack / move ----------

type targetSet uint8

const (
	targetsEnemyUnits targetSet = iota
	targetsEnemyBuildings
	targetsKnownResources
	targetsOwnBuildings
	targetsUnexplored
	targetsHome
)

var targetSetNames = map[string]targetSet{
	"enemy_units":     targetsEnemyUnits,
	"enemy_buildings": targetsEnemyBuildings,
	"known_resources": targetsKnownResources,
	"own_buildings":   targetsOwnBuildings,
	"unexplored":      targetsUnexplored,
	"home":            targetsHome,
}

type candidate struct {
	loc      ZoneRef
	unit     *UnitView
	building *BuildingView
}

// Scores every candidate of one kind with an expression and picks the best; nil when none reaches min_score
type targetPicker struct {
	set          targetSet
	unit_filter  unitFilter
	building_ids map[uint32]bool
	category     ResourceCategory
	score        amount
	min_score    amount
	has_min      bool
	together     bool
	// attack only: hit the chosen candidate itself (chasing it), or everything in its zone or space
	order attackOrderKind
	// attack only: share attackers out over the best candidates instead of all taking the best one
	distribute *distribution
}

// How attackers are shared over targets: each gets about enough to kill it within a hit, best-scoring targets first
type distribution struct {
	// expected damage one attacker deals a candidate per hit (evaluated per candidate)
	damage amount
	// extra share of attackers allowed per target, so a few misses or late arrivals still kill it
	overkill amount
	// follow-up targets each attacker queues behind its own
	queue int
}

type attackOrderKind uint8

const (
	attackOrderUnit attackOrderKind = iota
	attackOrderZone
	attackOrderSpace
)

func parseTargetPicker(raw map[string]any) (*targetPicker, error) {
	name, ok := raw["targets"].(string)
	if !ok {
		return nil, nil
	}
	set, ok := targetSetNames[name]
	if !ok {
		return nil, fmt.Errorf("unknown \"targets\" %q", name)
	}
	p := &targetPicker{set: set}
	var err error
	if p.unit_filter, err = parseUnitFilter(map[string]any{"unit_ids": raw["target_unit_ids"], "unit_types": raw["target_unit_types"]}); err != nil {
		return nil, err
	}
	if p.building_ids, err = parseIDSet(raw, "target_building_ids"); err != nil {
		return nil, err
	}
	if set == targetsKnownResources {
		if p.category, err = parseResourceCategory(raw["category"]); err != nil {
			return nil, fmt.Errorf("targets \"known_resources\": %v", err)
		}
	}
	if p.score, err = parseNumber(raw, "score", 0); err != nil {
		return nil, err
	}
	_, p.has_min = raw["min_score"]
	if p.min_score, err = parseNumber(raw, "min_score", 0); err != nil {
		return nil, err
	}
	p.together, _ = raw["together"].(bool)
	switch raw["order"] {
	case nil, "unit":
		p.order = attackOrderUnit
	case "zone":
		p.order = attackOrderZone
	case "space":
		p.order = attackOrderSpace
	default:
		return nil, fmt.Errorf("\"order\" must be \"unit\", \"zone\" or \"space\"")
	}
	if raw["distribute"] != nil {
		obj, ok := raw["distribute"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("\"distribute\" must be an object")
		}
		if p.order != attackOrderUnit {
			return nil, fmt.Errorf("\"distribute\" assigns single targets, so it needs \"order\": \"unit\"")
		}
		d := &distribution{queue: 2}
		if d.damage, err = parseNumber(obj, "damage", 1); err != nil {
			return nil, err
		}
		if d.overkill, err = parseNumber(obj, "overkill", 0); err != nil {
			return nil, err
		}
		if q, ok := obj["queue"].(float64); ok {
			d.queue = int(q)
		}
		p.distribute = d
	}
	return p, nil
}

func (p *targetPicker) candidates(view View, from ZoneRef) []candidate {
	out := make([]candidate, 0)
	switch p.set {
	case targetsEnemyUnits:
		for _, u := range view.VisibleEnemyUnits() {
			if len(p.unit_filter.unit_ids) == 0 && len(p.unit_filter.unit_types) == 0 || p.unit_filter.unit_ids[u.UnitID] || p.unit_filter.unit_types[u.Type] {
				out = append(out, candidate{loc: u.Location, unit: &u})
			}
		}
	case targetsEnemyBuildings, targetsOwnBuildings:
		buildings := view.KnownEnemyBuildings()
		if p.set == targetsOwnBuildings {
			buildings = view.Buildings()
		}
		for _, b := range buildings {
			if len(p.building_ids) == 0 || p.building_ids[b.BuildingID] {
				out = append(out, candidate{loc: b.Location, building: &b})
			}
		}
	case targetsKnownResources:
		for _, r := range view.KnownResources(p.category) {
			out = append(out, candidate{loc: r.Location})
		}
	case targetsUnexplored:
		if zones, ok := view.NearestUnexplored(from); ok {
			for _, z := range zones {
				out = append(out, candidate{loc: z})
			}
		}
	case targetsHome:
		if home, ok := homeLocation(view); ok {
			out = append(out, candidate{loc: home})
		}
	}
	return out
}

// best scores the candidates seen from `from`; target_distance and the "target" anchor refer to each in turn
func (p *targetPicker) best(view View, internals *Internals, from ZoneRef) (candidate, bool) {
	saved_target, saved_from, saved_h, saved_mh := internals.target, internals.target_from, internals.target_health, internals.target_max_health
	defer func() {
		internals.target, internals.target_from, internals.target_health, internals.target_max_health = saved_target, saved_from, saved_h, saved_mh
	}()
	min_score := p.min_score.float(view, internals)
	var best candidate
	best_score, found := 0.0, false
	for _, c := range p.candidates(view, from) {
		loc := c.loc
		internals.target, internals.target_from = &loc, &from
		internals.target_health, internals.target_max_health = c.health()
		score := p.score.float(view, internals)
		if p.has_min && score < min_score {
			continue
		}
		if !found || score > best_score || (score == best_score && zoneRefLess(c.loc, best.loc)) {
			best, best_score, found = c, score, true
		}
	}
	return best, found
}

// pick runs the picker for a group: with "together" every unit takes the target chosen from the group's center
// (units more than two spaces from it regroup first); otherwise each unit chooses from where it stands.
func (p *targetPicker) pick(view View, internals *Internals, units []UnitView, act func(u UnitView, c candidate, clear bool) Order) []Order {
	if p.distribute != nil {
		return p.distributed(view, internals, units, act)
	}
	orders := make([]Order, 0, len(units))
	if !p.together {
		for _, u := range units {
			if c, ok := p.best(view, internals, u.Location); ok {
				orders = append(orders, act(u, c, true))
			}
		}
		return orders
	}
	center, ok := groupCenter(units)
	if !ok {
		return orders
	}
	c, ok := p.best(view, internals, center)
	if !ok {
		return orders
	}
	for _, u := range units {
		if axialDistance(u.Location.Space, center.Space) > 2 {
			orders = append(orders, view.MoveOrder(u, center, true))
		} else {
			orders = append(orders, act(u, c, true))
		}
	}
	return orders
}

func attackCandidateOrder(view View, u UnitView, c candidate, order attackOrderKind, clear bool) Order {
	switch order {
	case attackOrderZone:
		return view.AttackZoneOrder(u, c.loc, clear)
	case attackOrderSpace:
		return view.AttackSpaceOrder(u, c.loc.Space, clear)
	}
	switch {
	case c.unit != nil:
		return view.AttackUnitOrder(u, *c.unit, clear)
	case c.building != nil:
		return view.AttackBuildingOrder(u, *c.building, clear)
	}
	return view.AttackZoneOrder(u, c.loc, clear)
}

func (c candidate) health() (float64, float64) {
	switch {
	case c.unit != nil:
		return c.unit.Health, c.unit.MaxHealth
	case c.building != nil:
		return c.building.Health, c.building.MaxHealth
	}
	return 0, 0
}

// distributed shares a group's attackers over the best candidates seen from its center: each candidate, best score
// first, gets its nearest free attackers until they would kill it in one hit (plus the overkill allowance); attackers
// left over go round the candidates again. Each attacker then queues the next candidates behind its own, so it moves on
// to a fresh target instead of piling onto one when its own dies. Units more than two spaces out regroup first.
func (p *targetPicker) distributed(view View, internals *Internals, units []UnitView, act func(u UnitView, c candidate, clear bool) Order) []Order {
	orders := make([]Order, 0, len(units))
	center, ok := groupCenter(units)
	if !ok {
		return orders
	}
	type target struct {
		c        candidate
		score    float64
		need     int
		assigned int
	}
	saved_target, saved_from, saved_h, saved_mh := internals.target, internals.target_from, internals.target_health, internals.target_max_health
	min_score := p.min_score.float(view, internals)
	targets := make([]*target, 0)
	for _, c := range p.candidates(view, center) {
		loc := c.loc
		internals.target, internals.target_from = &loc, &center
		internals.target_health, internals.target_max_health = c.health()
		score := p.score.float(view, internals)
		if p.has_min && score < min_score {
			continue
		}
		damage := max(0.01, p.distribute.damage.float(view, internals))
		overkill := max(0, p.distribute.overkill.float(view, internals))
		need := max(1, int(math.Ceil(max(internals.target_health, 0.01)*(1+overkill)/damage)))
		targets = append(targets, &target{c: c, score: score, need: need})
	}
	internals.target, internals.target_from, internals.target_health, internals.target_max_health = saved_target, saved_from, saved_h, saved_mh
	if len(targets) == 0 {
		return orders
	}
	sort.SliceStable(targets, func(i, j int) bool {
		if targets[i].score != targets[j].score {
			return targets[i].score > targets[j].score
		}
		return zoneRefLess(targets[i].c.loc, targets[j].c.loc)
	})
	free := make([]UnitView, 0, len(units))
	for _, u := range units {
		if axialDistance(u.Location.Space, center.Space) > 2 {
			orders = append(orders, view.MoveOrder(u, center, true))
		} else {
			free = append(free, u)
		}
	}
	assignment := make(map[uint64]int, len(free))
	take := func(t int) {
		best, best_d := -1, 0
		for i, u := range free {
			if d := locationDistance(u.Location, targets[t].c.loc); best < 0 || d < best_d {
				best, best_d = i, d
			}
		}
		assignment[free[best].InternalID] = t
		targets[t].assigned++
		free = append(free[:best], free[best+1:]...)
	}
	for t := range targets {
		for targets[t].assigned < targets[t].need && len(free) > 0 {
			take(t)
		}
	}
	for t := 0; len(free) > 0; t = (t + 1) % len(targets) {
		take(t)
	}
	// the attackers sharing a target queue different next targets, so they don't all pile onto one when it dies
	nth := make(map[int]int, len(targets))
	for _, u := range units {
		t, ok := assignment[u.InternalID]
		if !ok {
			continue
		}
		orders = append(orders, act(u, targets[t].c, true))
		offset := nth[t]
		nth[t]++
		for k := 1; k <= p.distribute.queue && k < len(targets); k++ {
			next := (t + offset + k) % len(targets)
			if next == t {
				next = (next + 1) % len(targets)
			}
			orders = append(orders, act(u, targets[next].c, false))
		}
	}
	return orders
}

// moveAction sends units to the best-scoring target of a picker
type moveAction struct {
	filtered
	picker   *targetPicker
	eligible []OrderKind
	max      amount
}

func (a *moveAction) ToOrders(view View, internals *Internals) []Order {
	units := a.filter.apply(eligibleUnits(view, a.eligible), anyUnit)
	if limit := a.max.int(view, internals); limit > 0 && len(units) > limit {
		units = units[:limit]
	}
	return a.picker.pick(view, internals, units, func(u UnitView, c candidate, clear bool) Order {
		return view.MoveOrder(u, c.loc, clear)
	})
}

func parseMove(raw map[string]any) (Action, error) {
	picker, err := parseTargetPicker(raw)
	if err != nil {
		return nil, err
	}
	if picker == nil {
		return nil, fmt.Errorf("move action requires \"targets\"")
	}
	if raw["order"] != nil || raw["distribute"] != nil {
		return nil, fmt.Errorf("\"order\" and \"distribute\" only apply to attack")
	}
	eligible, err := parseEligible(raw["eligible"])
	if err != nil {
		return nil, err
	}
	return &moveAction{picker: picker, eligible: eligible, max: parseMax(raw)}, nil
}

// target_distance / target_distance_home, only meaningful inside a picker's score
func targetDistanceCounter(name string) (counter, bool) {
	switch strings.TrimSpace(name) {
	case "target_distance":
		return func(_ View, i *Internals) float64 {
			if i.target == nil || i.target_from == nil {
				return 0
			}
			return float64(axialDistance(i.target_from.Space, i.target.Space))
		}, true
	case "target_health":
		return func(_ View, i *Internals) float64 { return i.target_health }, true
	case "target_max_health":
		return func(_ View, i *Internals) float64 { return i.target_max_health }, true
	case "target_distance_home":
		return func(v View, i *Internals) float64 {
			home, ok := homeLocation(v)
			if i.target == nil || !ok {
				return 0
			}
			return float64(axialDistance(home.Space, i.target.Space))
		}, true
	}
	return nil, false
}
