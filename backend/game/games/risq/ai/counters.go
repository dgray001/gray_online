package ai

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/dgray001/gray_online/util"
)

type comparison uint8

const (
	compareAtLeast comparison = iota
	compareAtMost
	compareEquals
)

var comparisonSuffixes = map[string]comparison{
	"_at_least": compareAtLeast,
	"_at_most":  compareAtMost,
	"_equals":   compareEquals,
}

type amount struct {
	value  float64
	expr   string
	query  counter
	impure bool
}

func parseAmount(raw any) (amount, error) {
	switch v := raw.(type) {
	case map[string]any:
		for _, name := range aggregateNames {
			if obj, ok := v[name].(map[string]any); ok && len(v) == 1 {
				query, impure, err := trackRandom(func() (counter, error) { return parseAggregate(name, obj) })
				return amount{query: query, impure: impure}, err
			}
		}
		return amount{}, fmt.Errorf("numeric query must be a count, sum, min or max object")
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return amount{}, fmt.Errorf("must be finite")
		}
		return amount{value: v}, nil
	case string:
		if err := util.ValidateExpr(v, nil, noteVarRead); err != nil {
			return amount{}, err
		}
		reads, _ := util.AnalyzeExpr(v)
		if reads.Impure {
			noteRandomUse()
		}
		return amount{expr: v, impure: reads.Impure}, nil
	}
	return amount{}, fmt.Errorf("must be a number, expression string or space query object")
}

// Reads an optional number-or-expression field, falling back to def when absent
func parseNumber(raw map[string]any, key string, def float64) (amount, error) {
	v, present := raw[key]
	if !present {
		return amount{value: def}, nil
	}
	a, err := parseAmount(v)
	if err != nil {
		return amount{}, fmt.Errorf("%q %v", key, err)
	}
	return a, nil
}

func evalGameVar(view View, internals *Internals, field string) (float64, error) {
	switch field {
	case "food":
		return internals.available(view, ResourceFood), nil
	case "wood":
		return internals.available(view, ResourceWood), nil
	case "stone":
		return internals.available(view, ResourceStone), nil
	case "gold":
		return internals.available(view, ResourceGold), nil
	case "population":
		current, _ := internals.population(view)
		return float64(current), nil
	case "population_limit":
		_, limit := internals.population(view)
		return float64(limit), nil
	case "population_max":
		return float64(view.MaxPopulation()), nil
	case "turn":
		return float64(view.TurnNumber()), nil
	case "map_size":
		return float64(view.MapSize()), nil
	case "num_players":
		return float64(view.NumPlayers()), nil
	}
	return 0, fmt.Errorf("unknown gameVar field %q", field)
}

func (a amount) resolve(view View, internals *Internals) (value float64, err error) {
	previous := internals.eval_errors
	defer func() {
		if err == nil && internals.eval_errors != previous {
			err = fmt.Errorf("nested expression evaluation failed")
		}
		if err == nil && (math.IsNaN(value) || math.IsInf(value, 0)) {
			err = fmt.Errorf("expression result must be finite")
		}
		if err != nil {
			internals.eval_errors++
		}
	}()
	if a.query != nil {
		return a.query(view, internals), nil
	}
	if a.expr == "" {
		return a.value, nil
	}
	return util.EvalExprWithFunctions(a.expr, nil, func(name string) (float64, error) {
		return internals.lookupVar(name), nil
	}, func(a string, b string, zone_steps bool) (float64, error) {
		return internals.distance(view, a, b, zone_steps)
	}, view.ConfigValue, func(field string) (float64, error) {
		return evalGameVar(view, internals, field)
	}, func(tech_id uint32) (bool, error) {
		return view.TechResearched(tech_id), nil
	}, internals.membership, internals.rng)
}

func (a amount) independentOf(role string) bool {
	if a.query != nil || a.impure || strings.Contains(a.expr, "distance(") {
		return false
	}
	reads, _ := util.AnalyzeExpr(a.expr)
	return !slices.Contains(reads.Roles, role) && !slices.ContainsFunc(reads.Vars, func(name string) bool { return strings.HasPrefix(name, role+".") })
}

// Resolves to a number, or 0 (with a one-time warning) when the expression can't be evaluated
func (a amount) float(view View, internals *Internals) float64 {
	v, err := a.resolve(view, internals)
	if err != nil {
		internals.warnOnce("expr:"+a.expr, fmt.Sprintf("ai expression %q: %v", a.expr, err))
		return 0
	}
	return v
}

func (a amount) int(view View, internals *Internals) int {
	return int(math.Round(a.float(view, internals)))
}

type countCondition struct {
	count  func(view View, internals *Internals) float64
	cmp    comparison
	amount amount
}

func (c *countCondition) Evaluate(view View, internals *Internals) bool {
	target, err := c.amount.resolve(view, internals)
	if err != nil {
		fmt.Fprintln(internals.diagnostics, "ai condition amount:", err)
		return false
	}
	n := c.count(view, internals)
	switch c.cmp {
	case compareAtLeast:
		return n >= target
	case compareAtMost:
		return n <= target
	default:
		return n == target
	}
}

func parseCountCondition(key string, value any) (Condition, error) {
	for suffix, cmp := range comparisonSuffixes {
		if key != "value"+suffix {
			continue
		}
		obj, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s must be an object", key)
		}
		count, err := parseValueCounter(obj)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", key, err)
		}
		a, err := parseAmount(obj["amount"])
		if err != nil {
			return nil, fmt.Errorf("%s: %v", key, err)
		}
		return &countCondition{count: count, cmp: cmp, amount: a}, nil
	}
	return nil, fmt.Errorf("unknown condition type %q", key)
}

type counter func(view View, internals *Internals) float64

var buildingStates = map[string]func(BuildingView) bool{
	"complete":           func(b BuildingView) bool { return !b.UnderConstruction },
	"under_construction": func(b BuildingView) bool { return b.UnderConstruction },
	"damaged":            func(b BuildingView) bool { return !b.UnderConstruction && b.Health < b.MaxHealth },
	"depleted":           func(b BuildingView) bool { return b.Gatherable && !b.UnderConstruction && b.ResourcesLeft <= 0 },
}
