package ai

import (
	"fmt"
	"math"
	"os"
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
	value float64
	expr  string
	query counter
}

var expressionVarNames = []string{
	"num_players", "enemies_found", "turn", "map_size", "land", "score", "best_enemy_score",
	"population", "population_limit", "food", "wood", "stone", "gold",
}

func parseAmount(raw any) (amount, error) {
	switch v := raw.(type) {
	case map[string]any:
		for name, parse := range map[string]func(map[string]any) (counter, error){"count_spaces": parseSpaceCounter, "closest_spaces": parseClosestSpaceCounter} {
			if obj, ok := v[name].(map[string]any); ok && len(v) == 1 {
				query, err := parse(obj)
				return amount{query: query}, err
			}
		}
		return amount{}, fmt.Errorf("numeric query must be a count_spaces or closest_spaces object")
	case float64:
		return amount{value: v}, nil
	case string:
		vars := make(map[string]float64, len(expressionVarNames))
		for _, name := range expressionVarNames {
			vars[name] = 1
		}
		// syntax check only: every var() resolves to 1, and a zero divisor from those stand-ins is not an error
		if _, err := util.EvalExprWithLookup(v, vars, func(name string) (float64, error) { noteVarRead(name); return 1, nil }); err != nil && err.Error() != "division by zero" {
			return amount{}, err
		}
		return amount{expr: v}, nil
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

func (a amount) resolve(view View, internals *Internals) (float64, error) {
	if a.query != nil {
		return a.query(view, internals), nil
	}
	if a.expr == "" {
		return a.value, nil
	}
	return util.EvalExprWithLookup(a.expr, expressionVars(view, internals), func(name string) (float64, error) {
		return internals.lookupVar(view, name), nil
	})
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

func expressionVars(view View, internals *Internals) map[string]float64 {
	current, limit := internals.population(view)
	return map[string]float64{
		"num_players":      float64(view.NumPlayers()),
		"enemies_found":    float64(view.EnemiesFound()),
		"turn":             float64(view.TurnNumber()),
		"map_size":         float64(view.MapSize()),
		"land":             float64(view.OwnedSpaces()),
		"score":            float64(view.Score()),
		"best_enemy_score": float64(view.BestEnemyScore()),
		"population":       float64(current),
		"population_limit": float64(limit),
		"food":             internals.available(view, ResourceFood),
		"wood":             internals.available(view, ResourceWood),
		"stone":            internals.available(view, ResourceStone),
		"gold":             internals.available(view, ResourceGold),
	}
}

type countCondition struct {
	count  func(view View, internals *Internals) float64
	cmp    comparison
	amount amount
}

func (c *countCondition) Evaluate(view View, internals *Internals) bool {
	target, err := c.amount.resolve(view, internals)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ai condition amount:", err)
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
		base, found := strings.CutSuffix(key, suffix)
		parse, known := counterParsers[base]
		if !found || !known {
			continue
		}
		obj, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s must be an object", key)
		}
		count, err := parse(obj)
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

func fixedCounter(c counter) func(map[string]any) (counter, error) {
	return func(map[string]any) (counter, error) { return c, nil }
}

var counterParsers map[string]func(obj map[string]any) (counter, error)

// Filled in init: these parsers reach parseAmount, which reads counterParsers, so a literal would be an initialization cycle
func init() {
	counterParsers = map[string]func(obj map[string]any) (counter, error){
		"turn":                fixedCounter(func(v View, _ *Internals) float64 { return float64(v.TurnNumber()) }),
		"map_size":            fixedCounter(func(v View, _ *Internals) float64 { return float64(v.MapSize()) }),
		"num_players":         fixedCounter(func(v View, _ *Internals) float64 { return float64(v.NumPlayers()) }),
		"enemies_found":       fixedCounter(func(v View, _ *Internals) float64 { return float64(v.EnemiesFound()) }),
		"land":                fixedCounter(func(v View, _ *Internals) float64 { return float64(v.OwnedSpaces()) }),
		"score_lead":          fixedCounter(func(v View, _ *Internals) float64 { return float64(v.Score() - v.BestEnemyScore()) }),
		"population_headroom": fixedCounter(populationHeadroom),
		"resource":            parseResourceCounter,
		"resource_remaining":  parseResourceRemainingCounter,
		"population":          parsePopulationCounter,
		"idle_units":          parseIdleUnitsCounter, "enemy_units_visible": parseEnemyUnitsCounter,
		"enemy_units_unidentified": parseUnidentifiedUnitsCounter,
		"enemy_buildings_known":    parseEnemyBuildingsCounter,
		"building_count":           parseBuildingCounter,
		"foundation_count":         parseFoundationCounter,
		"foundation_space_x":       parseFoundationSpaceCounter(true),
		"foundation_space_y":       parseFoundationSpaceCounter(false),
		"bucket_size":              parseBucketSizeCounter,
		"economic_producers":       fixedCounter(economicProducers),
		"available_gatherers":      fixedCounter(availableGatherers),
		"count_spaces":             parseSpaceCounter,
		"closest_spaces":           parseClosestSpaceCounter,
		"base_spaces":              parseBaseSpacesCounter,
	}
}

func availableGatherers(view View, _ *Internals) float64 {
	count := 0
	for _, unit := range view.EligibleUnits(OrderKindGather) {
		if unit.Kind == UnitEconomic && unit.GarrisonedIn == nil {
			count++
		}
	}
	return float64(count)
}

func economicProducers(view View, _ *Internals) float64 {
	count := 0
	for _, building := range view.Buildings() {
		for _, option := range building.Producibles {
			if option.Kind == ProducibleUnit && option.UnitType == UnitTypeEconomic {
				count++
				break
			}
		}
	}
	return float64(count)
}

func populationHeadroom(view View, internals *Internals) float64 {
	current, limit := internals.population(view)
	return float64(limit - current)
}

func parseResourceCounter(obj map[string]any) (counter, error) {
	category, err := parseResourceCategory(obj["category"])
	if err != nil {
		return nil, err
	}
	return func(v View, i *Internals) float64 { return i.available(v, category) }, nil
}

func parseResourceRemainingCounter(obj map[string]any) (counter, error) {
	category, err := parseResourceCategory(obj["category"])
	if err != nil {
		return nil, err
	}
	near, err := parseNearFilter(obj)
	if err != nil {
		return nil, err
	}
	return func(v View, i *Internals) float64 {
		total := 0.0
		for _, r := range v.KnownResources(category) {
			if near.contains(v, i, r.Location) {
				total += r.AmountLeft
			}
		}
		return total
	}, nil
}

func parsePopulationCounter(obj map[string]any) (counter, error) {
	filter, err := parseUnitFilter(obj)
	if err != nil {
		return nil, err
	}
	near, err := parseNearFilter(obj)
	if err != nil {
		return nil, err
	}
	bucket, _ := obj["in_bucket"].(string)
	return func(v View, i *Internals) float64 {
		n := 0
		for _, u := range filter.apply(v.Units(), anyUnit) {
			if (bucket == "" || i.inBucket(bucket, u.InternalID)) && near.contains(v, i, u.Location) {
				n++
			}
		}
		return float64(n)
	}, nil
}

func parseIdleUnitsCounter(obj map[string]any) (counter, error) {
	filter, err := parseUnitFilter(obj)
	if err != nil {
		return nil, err
	}
	bucket, _ := obj["in_bucket"].(string)
	return func(v View, i *Internals) float64 {
		n := 0
		for _, u := range filter.apply(v.IdleUnits(), anyUnit) {
			if bucket == "" || i.inBucket(bucket, u.InternalID) {
				n++
			}
		}
		return float64(n)
	}, nil
}

func parseEnemyUnitsCounter(obj map[string]any) (counter, error) {
	filter, err := parseUnitFilter(obj)
	if err != nil {
		return nil, err
	}
	near, err := parseNearFilter(obj)
	if err != nil {
		return nil, err
	}
	return func(v View, i *Internals) float64 {
		n := 0
		for _, u := range filter.apply(v.VisibleEnemyUnits(), anyUnit) {
			if near.contains(v, i, u.Location) {
				n++
			}
		}
		return float64(n)
	}, nil
}

func parseUnidentifiedUnitsCounter(obj map[string]any) (counter, error) {
	near, err := parseNearFilter(obj)
	if err != nil {
		return nil, err
	}
	return func(v View, i *Internals) float64 {
		count := 0
		for _, space := range v.AllSpaces() {
			if near.contains(v, i, ZoneRef{Space: space.Space}) {
				count += unidentifiedUnitCount(v, space)
			}
		}
		return float64(count)
	}, nil
}

func parseEnemyBuildingsCounter(obj map[string]any) (counter, error) {
	ids, err := parseIDSet(obj, "building_ids")
	if err != nil {
		return nil, err
	}
	near, err := parseNearFilter(obj)
	if err != nil {
		return nil, err
	}
	return func(v View, i *Internals) float64 {
		n := 0
		for _, b := range v.KnownEnemyBuildings() {
			if (len(ids) == 0 || ids[b.BuildingID]) && near.contains(v, i, b.Location) {
				n++
			}
		}
		return float64(n)
	}, nil
}

var buildingStates = map[string]func(BuildingView) bool{
	"complete":           func(b BuildingView) bool { return !b.UnderConstruction },
	"under_construction": func(b BuildingView) bool { return b.UnderConstruction },
	"damaged":            func(b BuildingView) bool { return !b.UnderConstruction && b.Health < b.MaxHealth },
	"depleted":           func(b BuildingView) bool { return b.Gatherable && !b.UnderConstruction && b.ResourcesLeft <= 0 },
}

func parseBuildingCounter(obj map[string]any) (counter, error) {
	ids, err := parseIDSet(obj, "building_ids")
	if err != nil {
		return nil, err
	}
	state := func(BuildingView) bool { return true }
	if s, ok := obj["state"].(string); ok {
		if state, ok = buildingStates[s]; !ok {
			return nil, fmt.Errorf("unknown building state %q", s)
		}
	}
	return func(v View, _ *Internals) float64 {
		n := 0
		for _, b := range v.Buildings() {
			if (len(ids) == 0 || ids[b.BuildingID]) && state(b) {
				n++
			}
		}
		return float64(n)
	}, nil
}

func parseFoundationCounter(obj map[string]any) (counter, error) {
	ids, err := parseIDSet(obj, "building_ids")
	if err != nil {
		return nil, err
	}
	without_builders, _ := obj["without_builders"].(bool)
	return func(v View, _ *Internals) float64 {
		n := 0
		for _, f := range v.Foundations() {
			if (len(ids) == 0 || ids[f.BuildingID]) && (!without_builders || f.Builders == 0) {
				n++
			}
		}
		return float64(n)
	}, nil
}

// Space coordinate of the matching foundation with the lowest zone ref, or 0 when none matches
func parseFoundationSpaceCounter(x_axis bool) func(map[string]any) (counter, error) {
	return func(obj map[string]any) (counter, error) {
		ids, err := parseIDSet(obj, "building_ids")
		if err != nil {
			return nil, err
		}
		without_builders, _ := obj["without_builders"].(bool)
		return func(v View, _ *Internals) float64 {
			var best ZoneRef
			found := false
			for _, f := range v.Foundations() {
				if (len(ids) == 0 || ids[f.BuildingID]) && (!without_builders || f.Builders == 0) && (!found || zoneRefLess(f.Location, best)) {
					best, found = f.Location, true
				}
			}
			if x_axis {
				return float64(best.Space.X)
			}
			return float64(best.Space.Y)
		}, nil
	}
}

func parseBucketSizeCounter(obj map[string]any) (counter, error) {
	bucket, ok := obj["bucket"].(string)
	if !ok {
		return nil, fmt.Errorf("bucket_size requires a string \"bucket\"")
	}
	return func(_ View, i *Internals) float64 {
		if b := i.Buckets[bucket]; b != nil {
			return float64(len(b.Members))
		}
		return 0
	}, nil
}

func parseBaseSpacesCounter(obj map[string]any) (counter, error) {
	near, err := parseNearFilter(obj)
	if err != nil {
		return nil, err
	}
	return func(v View, i *Internals) float64 {
		count := 0
		for _, s := range baseSpaces(v) {
			if near.contains(v, i, ZoneRef{Space: s}) {
				count++
			}
		}
		return float64(count)
	}, nil
}
