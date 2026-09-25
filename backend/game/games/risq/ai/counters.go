package ai

import (
	"fmt"
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
}

var expressionVarNames = []string{
	"num_players", "enemies_found", "turn", "land", "score", "best_enemy_score",
	"population", "population_limit", "food", "wood", "stone", "gold",
}

func parseAmount(raw any) (amount, error) {
	switch v := raw.(type) {
	case float64:
		return amount{value: v}, nil
	case string:
		vars := make(map[string]float64, len(expressionVarNames))
		for _, name := range expressionVarNames {
			vars[name] = 1
		}
		if _, err := util.EvalExpr(v, vars); err != nil {
			return amount{}, err
		}
		return amount{expr: v}, nil
	}
	return amount{}, fmt.Errorf("\"amount\" must be a number or expression string")
}

func (a amount) resolve(view View, internals *Internals) (float64, error) {
	if a.expr == "" {
		return a.value, nil
	}
	return util.EvalExpr(a.expr, expressionVars(view, internals))
}

func expressionVars(view View, internals *Internals) map[string]float64 {
	current, limit := internals.population(view)
	return map[string]float64{
		"num_players":      float64(view.NumPlayers()),
		"enemies_found":    float64(view.EnemiesFound()),
		"turn":             float64(view.TurnNumber()),
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

var counterParsers = map[string]func(obj map[string]any) (counter, error){
	"turn":                fixedCounter(func(v View, _ *Internals) float64 { return float64(v.TurnNumber()) }),
	"num_players":         fixedCounter(func(v View, _ *Internals) float64 { return float64(v.NumPlayers()) }),
	"enemies_found":       fixedCounter(func(v View, _ *Internals) float64 { return float64(v.EnemiesFound()) }),
	"land":                fixedCounter(func(v View, _ *Internals) float64 { return float64(v.OwnedSpaces()) }),
	"score_lead":          fixedCounter(func(v View, _ *Internals) float64 { return float64(v.Score() - v.BestEnemyScore()) }),
	"population_headroom": fixedCounter(populationHeadroom),
	"resource":            parseResourceCounter,
	"population":          parsePopulationCounter,
	"idle_units":          parseIdleUnitsCounter,
	"enemy_units_visible": parseEnemyUnitsCounter,
	"building_count":      parseBuildingCounter,
	"foundation_count":    parseFoundationCounter,
	"bucket_size":         parseBucketSizeCounter,
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

func parsePopulationCounter(obj map[string]any) (counter, error) {
	filter, err := parseUnitFilter(obj)
	if err != nil {
		return nil, err
	}
	return func(v View, _ *Internals) float64 { return float64(len(filter.apply(v.Units(), anyUnit))) }, nil
}

func parseIdleUnitsCounter(obj map[string]any) (counter, error) {
	filter, err := parseUnitFilter(obj)
	if err != nil {
		return nil, err
	}
	return func(v View, _ *Internals) float64 { return float64(len(filter.apply(v.IdleUnits(), anyUnit))) }, nil
}

func parseEnemyUnitsCounter(obj map[string]any) (counter, error) {
	filter, err := parseUnitFilter(obj)
	if err != nil {
		return nil, err
	}
	within, has_within := obj["within"].(float64)
	return func(v View, _ *Internals) float64 {
		home, has_home := homeLocation(v)
		n := 0
		for _, u := range filter.apply(v.VisibleEnemyUnits(), anyUnit) {
			if !has_within || (has_home && axialDistance(home.Space, u.Location.Space) <= int(within)) {
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
