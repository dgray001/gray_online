package ai

import "fmt"

type exploreAnchor uint8

const (
	exploreAnchorSelf exploreAnchor = iota
	exploreAnchorHome
	exploreAnchorCenter
)

type exploreAction struct {
	filtered
	max          amount
	min_distance amount
	include_fog  amount
	anchor       exploreAnchor
}

func (a *exploreAction) ToOrders(view View, internals *Internals) []Order {
	limit := a.max.int(view, internals)
	min_distance := max(0, a.min_distance.int(view, internals))
	include_fog := a.include_fog.float(view, internals)
	orders := make([]Order, 0)
	var anchor ZoneRef
	fixed := a.anchor != exploreAnchorSelf
	if a.anchor == exploreAnchorHome {
		home, ok := homeLocation(view)
		if !ok {
			return orders
		}
		anchor = home
	}
	claimed := make(map[ZoneRef]bool)
	for _, u := range view.Units() {
		if u.CurrentOrder != nil && u.CurrentOrder.Kind == OrderKindMove && u.CurrentOrder.TargetZone != nil {
			claimed[*u.CurrentOrder.TargetZone] = true
		}
	}
	for _, u := range a.filter.apply(view.IdleUnits(), anyUnit) {
		if limit > 0 && len(orders) >= limit {
			break
		}
		from := u.Location
		if fixed {
			from = anchor
		}
		candidates, ok := nearestExplorationTargets(view, internals, from, min_distance, include_fog)
		if !ok {
			continue
		}
		available := make([]ZoneRef, 0, len(candidates))
		for _, c := range candidates {
			if !claimed[c] {
				available = append(available, c)
			}
		}
		if len(available) == 0 {
			available = candidates
		}
		target, ok := randomNearestSpace(view, internals, u.Location.Space, available)
		if !ok {
			continue
		}
		claimed[target] = true
		orders = append(orders, view.MoveOrder(u, target, true))
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
		if ok && (!found || score > best_score || (score == best_score && candidateLess(c, best))) {
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
	for _, key := range []string{"var", "var_x", "var_y", "var_score"} {
		if value, present := raw[key]; present {
			if name, ok := value.(string); !ok || !validVarName(name) {
				return nil, fmt.Errorf("%q must be a variable name", key)
			}
		}
	}
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
	if value, present := raw["options"]; present {
		rawOpts, ok := value.([]any)
		if !ok || len(rawOpts) == 0 {
			return nil, fmt.Errorf("options must be a nonempty array of pickers")
		}
		if _, has_targets := raw["targets"]; has_targets {
			return nil, fmt.Errorf("options and targets are mutually exclusive")
		}
		for _, o := range rawOpts {
			optMap, ok := o.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("options entries must be picker objects")
			}
			picker, err := parseTargetPicker(optMap)
			if err != nil {
				return nil, err
			}
			if picker == nil {
				return nil, fmt.Errorf("options entries require targets")
			}
			options = append(options, &selectSpaceOption{picker: picker})
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
