package ai

import (
	"fmt"
	"strings"

	"github.com/dgray001/gray_online/util"
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
	anchorCoordinate
)

type anchor struct {
	kind                       anchorKind
	bucket                     string
	coordinate_x, coordinate_y amount
}

func parseAnchor(raw any) (anchor, error) {
	if raw == nil {
		return anchor{kind: anchorHome}, nil
	}
	if obj, ok := raw.(map[string]any); ok {
		x, x_err := parseAmount(obj["x"])
		y, y_err := parseAmount(obj["y"])
		if x_err != nil || y_err != nil {
			return anchor{}, fmt.Errorf("coordinate reference requires x and y as numbers or expressions")
		}
		return anchor{kind: anchorCoordinate, coordinate_x: x, coordinate_y: y}, nil
	}
	name, ok := raw.(string)
	if !ok || name == "" {
		return anchor{}, fmt.Errorf("\"from\" must be a coordinate object, \"home\", \"target\", \"unit\" or a bucket name")
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
	case anchorCoordinate:
		return ZoneRef{Space: Coordinate{X: a.coordinate_x.int(view, internals), Y: a.coordinate_y.int(view, internals)}}, true
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
		return groupCenter(view, members)
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
func groupCenter(view View, units []UnitView) (ZoneRef, bool) {
	if len(units) == 0 {
		return ZoneRef{}, false
	}
	best, best_sum := units[0].Location, -1
	for _, u := range units {
		sum := 0
		for _, other := range units {
			sum += view.SpaceDistance(u.Location.Space, other.Location.Space)
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
	return ok && view.SpaceDistance(center.Space, loc.Space) <= f.within
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
	targetsRetreat
	targetsUnidentifiedUnits
	targetsClosestSpaces
	targetsCoordinate
)

var targetSetNames = map[string]targetSet{
	"enemy_units":        targetsEnemyUnits,
	"enemy_buildings":    targetsEnemyBuildings,
	"known_resources":    targetsKnownResources,
	"own_buildings":      targetsOwnBuildings,
	"unexplored":         targetsUnexplored,
	"home":               targetsHome,
	"retreat":            targetsRetreat,
	"unidentified_units": targetsUnidentifiedUnits,
	"closest_spaces":     targetsClosestSpaces,
	"coordinate":         targetsCoordinate,
}

type candidate struct {
	loc      ZoneRef
	unit     *UnitView
	building *BuildingView
}

// Scores every candidate of one kind with an expression and picks the best; nil when none reaches min_score
type targetPicker struct {
	set          targetSet
	to           string
	spaces       *spaceQuery
	coord        anchor
	unit_filter  unitFilter
	building_ids map[uint32]bool
	category     ResourceCategory
	score        amount
	min_score    amount
	has_min      bool
	together     bool
	// attack only: hit the chosen candidate itself (chasing it), or everything in its zone or space
	order attackOrderKind
	// share units out over the candidates (each unit scores them itself) instead of every unit taking the best one
	spread *spreadConfig
	// closest_spaces only: stand on the edge zone facing this anchor when it is the neighboring space, which gives full sight of it
	face *anchor
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
	if to, ok := raw["to"].(string); ok {
		p.to = to
	}
	var err error
	if set == targetsClosestSpaces {
		if p.spaces, err = parseSpaceQuery(raw); err != nil {
			return nil, err
		}
		if raw["face"] != nil {
			face, err := parseAnchor(raw["face"])
			if err != nil {
				return nil, err
			}
			p.face = &face
		}
	}
	if set == targetsCoordinate {
		if p.coord, err = parseAnchor(raw["coordinate"]); err != nil {
			return nil, err
		}
	}
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
		return nil, fmt.Errorf("\"distribute\" was replaced by \"spread\" (capacity, unit_score, unit_order, queue, overflow)")
	}
	if raw["spread"] != nil {
		if p.spread, err = parseSpread(raw["spread"]); err != nil {
			return nil, err
		}
	}
	return p, nil
}

func retreatLocation(view View, internals *Internals, from ZoneRef, to string) (ZoneRef, bool) {
	if to == "home" {
		if h, ok := homeLocation(view); ok {
			return h, true
		}
		if internals.start != nil {
			return *internals.start, true
		}
		return ZoneRef{}, false
	}
	bases := baseSpaces(view)
	for _, s := range bases {
		if s == from.Space {
			return ZoneRef{}, false
		}
	}
	home, has_home := homeLocation(view)
	best, bestD, best_home_distance := Coordinate{}, -1, 0
	for _, s := range bases {
		d, home_distance := view.SpaceDistance(from.Space, s), 0
		if has_home {
			home_distance = view.SpaceDistance(home.Space, s)
		}
		closer := d < bestD || (d == bestD && (home_distance < best_home_distance ||
			(home_distance == best_home_distance && util.Pair(s.X, s.Y) < util.Pair(best.X, best.Y))))
		if bestD == -1 || closer {
			bestD, best, best_home_distance = d, s, home_distance
		}
	}
	if bestD != -1 {
		return ZoneRef{Space: best}, true
	}
	if h, ok := homeLocation(view); ok {
		return h, true
	}
	if internals.start != nil {
		return *internals.start, true
	}
	return ZoneRef{}, false
}

// The space's edge zone toward the face anchor (a zone's coordinate is the direction of the neighbor it borders), else its center
func (p *targetPicker) edgeFacing(view View, internals *Internals, space Coordinate) ZoneRef {
	loc := ZoneRef{Space: space}
	if p.face == nil {
		return loc
	}
	if target, ok := p.face.location(view, internals); ok && view.SpaceDistance(space, target.Space) == 1 {
		direction := Coordinate{X: target.Space.X - space.X, Y: target.Space.Y - space.Y}
		if util.AbsInt(direction.X) <= 1 && util.AbsInt(direction.Y) <= 1 && direction.X != direction.Y {
			loc.Zone = direction
		}
	}
	return loc
}

func (p *targetPicker) candidates(view View, internals *Internals, from ZoneRef) []candidate {
	out := make([]candidate, 0)
	switch p.set {
	case targetsClosestSpaces:
		for _, space := range p.spaces.closest(view, internals, &from) {
			out = append(out, candidate{loc: p.edgeFacing(view, internals, space)})
		}
	case targetsEnemyUnits:
		for _, u := range view.VisibleEnemyUnits() {
			if p.unit_filter.matches(u) {
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
	case targetsRetreat:
		if loc, ok := retreatLocation(view, internals, from, p.to); ok {
			out = append(out, candidate{loc: loc})
		}
	case targetsUnidentifiedUnits:
		for _, space := range view.AllSpaces() {
			if unidentifiedUnitCount(view, space) > 0 {
				out = append(out, candidate{loc: ZoneRef{Space: space.Space}})
			}
		}
	case targetsCoordinate:
		if loc, ok := p.coord.location(view, internals); ok {
			out = append(out, candidate{loc: loc})
		}
	}
	return out
}

// best scores the candidates seen from `from`; target_distance and the "target" anchor refer to each in turn
func (p *targetPicker) best(view View, internals *Internals, from ZoneRef) (candidate, float64, bool) {
	saved_target, saved_from, saved_h, saved_mh := internals.target, internals.target_from, internals.target_health, internals.target_max_health
	defer func() {
		internals.target, internals.target_from, internals.target_health, internals.target_max_health = saved_target, saved_from, saved_h, saved_mh
	}()
	min_score := p.min_score.float(view, internals)
	var best candidate
	best_score, found := 0.0, false
	for _, c := range p.candidates(view, internals, from) {
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
	return best, best_score, found
}

// pick runs the picker for a group: with "together" every unit takes the target chosen from the group's center
// (units more than two spaces from it regroup first); otherwise each unit chooses from where it stands.
func (p *targetPicker) pick(view View, internals *Internals, units []UnitView, act func(u UnitView, c candidate, clear bool) Order) []Order {
	if p.spread != nil {
		return p.assign(view, internals, units, act)
	}
	orders := make([]Order, 0, len(units))
	if !p.together {
		for _, u := range units {
			if c, _, ok := p.best(view, internals, u.Location); ok {
				orders = append(orders, act(u, c, true))
			}
		}
		return orders
	}
	center, ok := groupCenter(view, units)
	if !ok {
		return orders
	}
	c, _, ok := p.best(view, internals, center)
	if !ok {
		return orders
	}
	for _, u := range units {
		if view.SpaceDistance(u.Location.Space, center.Space) > 2 {
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
	if raw["order"] != nil {
		return nil, fmt.Errorf("\"order\" only applies to attack")
	}
	eligible, err := parseEligible(raw["eligible"])
	if err != nil {
		return nil, err
	}
	return &moveAction{picker: picker, eligible: eligible, max: parseMax(raw)}, nil
}

type selectSpaceOption struct {
	picker *targetPicker
}

type selectSpaceAction struct {
	options   []*selectSpaceOption
	from      anchor
	var_x     string
	var_y     string
	var_score string
	min_score amount
	has_min   bool
	persist   bool
}

func (a *selectSpaceAction) ToOrders(view View, internals *Internals) []Order {
	from, ok := a.from.location(view, internals)
	if !ok {
		return nil
	}
	var best candidate
	best_score, found := 0.0, false
	for _, opt := range a.options {
		if opt.picker == nil {
			continue
		}
		c, score, ok := opt.picker.best(view, internals, from)
		if ok && (!found || score > best_score || (score == best_score && zoneRefLess(c.loc, best.loc))) {
			best, best_score, found = c, score, true
		}
	}
	if !found {
		return nil
	}
	if a.has_min && best_score < a.min_score.float(view, internals) {
		return nil
	}
	vars := internals.turn_vars
	if a.persist {
		if internals.vars == nil {
			internals.vars = make(map[string]float64)
		}
		vars = internals.vars
	} else if internals.turn_vars == nil {
		internals.turn_vars = make(map[string]float64)
		vars = internals.turn_vars
	}
	if a.var_x != "" {
		vars[a.var_x] = float64(best.loc.Space.X)
	}
	if a.var_y != "" {
		vars[a.var_y] = float64(best.loc.Space.Y)
	}
	if a.var_score != "" {
		vars[a.var_score] = best_score
	}
	return nil
}

func parseSelectSpace(raw map[string]any) (Action, error) {
	from, err := parseAnchor(raw["from"])
	if err != nil {
		return nil, err
	}
	persist, _ := raw["persist"].(bool)
	var_x, _ := raw["var_x"].(string)
	var_y, _ := raw["var_y"].(string)
	var_score, _ := raw["var_score"].(string)
	_, has_min := raw["min_score"]
	min_score, err := parseNumber(raw, "min_score", 0)
	if err != nil {
		return nil, err
	}
	if name, ok := raw["var"].(string); ok && name != "" {
		if var_x == "" {
			var_x = name + "_x"
		}
		if var_y == "" {
			var_y = name + "_y"
		}
	}
	if var_x != "" {
		noteVarSet(var_x)
	}
	if var_y != "" {
		noteVarSet(var_y)
	}
	if var_score != "" {
		noteVarSet(var_score)
	}
	var options []*selectSpaceOption
	if rawOpts, ok := raw["options"].([]any); ok {
		for _, o := range rawOpts {
			optMap, ok := o.(map[string]any)
			if !ok {
				continue
			}
			picker, err := parseTargetPicker(optMap)
			if err != nil {
				return nil, err
			}
			if picker != nil {
				options = append(options, &selectSpaceOption{picker: picker})
			}
		}
	} else {
		picker, err := parseTargetPicker(raw)
		if err != nil {
			return nil, err
		}
		if picker != nil {
			options = append(options, &selectSpaceOption{picker: picker})
		}
	}
	return &selectSpaceAction{
		options:   options,
		from:      from,
		var_x:     var_x,
		var_y:     var_y,
		var_score: var_score,
		min_score: min_score,
		has_min:   has_min,
		persist:   persist,
	}, nil
}

func (i *Internals) targetSpace() Coordinate {
	if i.target == nil {
		return Coordinate{}
	}
	return i.target.Space
}

func spaceVision(view View, space Coordinate) uint8 {
	for _, info := range view.AllSpaces() {
		if info.Space == space {
			return info.Vision
		}
	}
	return 0
}

// target_distance / target_distance_home, only meaningful inside a picker's score
func targetDistanceCounter(name string) (counter, bool) {
	switch strings.TrimSpace(name) {
	case "target_distance":
		return func(v View, i *Internals) float64 {
			if i.target == nil || i.target_from == nil {
				return 0
			}
			return float64(v.SpaceDistance(i.target_from.Space, i.target.Space))
		}, true
	case "unit_distance":
		return func(v View, i *Internals) float64 {
			if i.target == nil || i.unit == nil {
				return 0
			}
			return float64(v.LocationDistance(*i.unit, *i.target))
		}, true
	case "unit_health":
		return func(_ View, i *Internals) float64 { return i.unit_health }, true
	case "unit_id":
		return func(_ View, i *Internals) float64 { return i.unit_id }, true
	case "unit_stamina":
		return func(_ View, i *Internals) float64 { return i.unit_stamina }, true
	case "target_assigned":
		return func(_ View, i *Internals) float64 { return i.target_assigned }, true
	case "assign_round":
		return func(_ View, i *Internals) float64 { return i.assign_round }, true
	case "target_id":
		return func(_ View, i *Internals) float64 { return i.target_id }, true
	case "target_health":
		return func(_ View, i *Internals) float64 { return i.target_health }, true
	case "target_max_health":
		return func(_ View, i *Internals) float64 { return i.target_max_health }, true
	case "target_own_buildings":
		return func(v View, i *Internals) float64 { return float64(v.OwnBuildingsIn(i.targetSpace())) }, true
	case "target_enemy_distance":
		return func(v View, i *Internals) float64 { return float64(max(0, v.EnemyDistance(i.targetSpace()))) }, true
	case "target_vision":
		return func(v View, i *Internals) float64 { return float64(spaceVision(v, i.targetSpace())) }, true
	case "target_region_progress":
		return func(v View, i *Internals) float64 { return v.RegionProgress(i.targetSpace()) }, true
	case "enemy_distance":
		return func(v View, _ *Internals) float64 {
			home, ok := homeLocation(v)
			return float64(max(0, v.EnemyDistance(home.Space))) * boolNumber(ok)
		}, true
	case "target_distance_home":
		return func(v View, i *Internals) float64 {
			home, ok := homeLocation(v)
			if i.target == nil || !ok {
				return 0
			}
			return float64(v.SpaceDistance(home.Space, i.target.Space))
		}, true
	}
	return nil, false
}
