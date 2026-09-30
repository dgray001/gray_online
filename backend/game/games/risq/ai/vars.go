package ai

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

// setVarAction stores a number for later rules to read with var(name): in the per-turn map (cleared
// every turn) or, with "persist", in the script's global map that survives between turns.
type setVarAction struct {
	name    string
	value   amount
	persist bool
}

// Load-time bookkeeping for var() names: every name read and every name set_var writes
var (
	var_names_mu  sync.Mutex
	var_names_log *varNameLog
)

type varNameLog struct {
	read map[string]bool
	set  map[string]bool
}

func noteVarRead(name string) {
	if var_names_log != nil {
		var_names_log.read[name] = true
	}
}

func noteVarSet(name string) {
	if var_names_log != nil {
		var_names_log.set[name] = true
	}
}

// Runs parse with var() bookkeeping on, then reports names that are neither built-in nor ever set
func withVarNameCheck(parse func() error) error {
	var_names_mu.Lock()
	defer var_names_mu.Unlock()
	var_names_log = &varNameLog{read: map[string]bool{}, set: map[string]bool{}}
	defer func() { var_names_log = nil }()
	if err := parse(); err != nil {
		return err
	}
	for name := range var_names_log.read {
		if var_names_log.set[name] {
			continue
		}
		if _, err := builtinCounter(name); err != nil {
			fmt.Fprintf(os.Stderr, "ai config: var(%s) is not a built-in and no set_var sets it; it will read 0\n", name)
		}
	}
	return nil
}

func (a *setVarAction) ToOrders(view View, internals *Internals) []Order {
	v := a.value.float(view, internals)
	if a.persist {
		if internals.vars == nil {
			internals.vars = make(map[string]float64)
		}
		internals.vars[a.name] = v
	} else {
		if internals.turn_vars == nil {
			internals.turn_vars = make(map[string]float64)
		}
		internals.turn_vars[a.name] = v
	}
	return nil
}

func parseSetVar(raw map[string]any) (Action, error) {
	name, ok := raw["name"].(string)
	if !ok || !validVarName(name) {
		return nil, fmt.Errorf("set_var requires a \"name\" of letters, digits and underscores")
	}
	value, err := parseAmount(raw["value"])
	if err != nil {
		return nil, fmt.Errorf("set_var %q: \"value\" %v", name, err)
	}
	persist, _ := raw["persist"].(bool)
	noteVarSet(name)
	return &setVarAction{name: name, value: value, persist: persist}, nil
}

func validVarName(name string) bool {
	if name == "" {
		return false
	}
	for i, c := range name {
		letter := c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		if !letter && (i == 0 || c < '0' || c > '9') {
			return false
		}
	}
	return true
}

// var(name) resolution: this turn's variables, then global ones, then the built-in counters; anything else is 0
func (i *Internals) lookupVar(view View, name string) float64 {
	if v, ok := i.turn_vars[name]; ok {
		return v
	}
	if v, ok := i.vars[name]; ok {
		return v
	}
	c, cached := i.builtin_vars[name]
	if !cached {
		// not a built-in: a script variable that hasn't been set yet (unknown names are reported at load)
		c, _ = builtinCounter(name)
		if i.builtin_vars == nil {
			i.builtin_vars = make(map[string]counter)
		}
		i.builtin_vars[name] = c
	}
	if c == nil {
		return 0
	}
	return c(view, i)
}

func (i *Internals) warnOnce(key string, message string) {
	if i.warned == nil {
		i.warned = make(map[string]bool)
	}
	if !i.warned[key] {
		i.warned[key] = true
		fmt.Fprintln(os.Stderr, message)
	}
}

// Left out of var() because a script can derive them: population_headroom is population_limit - population,
// score_lead is score - best_enemy_score, resource_food is food; value only makes sense as a condition
var derivableCounters = map[string]bool{"population_headroom": true, "score_lead": true, "resource": true, "value": true}

// Built-in var() names mirror the count conditions, with their filters spelled out after underscores:
// population_13, population_infantry, idle_units_1, enemy_units_visible_13_within_2, building_count_23_complete,
// foundation_count_2_without_builders, resource_food, resource_remaining_food_within_1, bucket_size_<bucket>,
// tech_researched_5, resource_available_gold, plus the plain expression variables (turn, food, population_limit, ...)
func builtinCounter(name string) (counter, error) {
	if c, ok := targetDistanceCounter(name); ok {
		return c, nil
	}
	if name == "population_max" {
		return func(v View, _ *Internals) float64 { return float64(v.MaxPopulation()) }, nil
	}
	if c, ok, err := costCounter(name); ok {
		return c, err
	}
	for _, v := range expressionVarNames {
		if name == v {
			return func(view View, internals *Internals) float64 { return expressionVars(view, internals)[v] }, nil
		}
	}
	base := ""
	for candidate := range allCounterParsers() {
		if derivableCounters[candidate] {
			continue
		}
		if (name == candidate || strings.HasPrefix(name, candidate+"_")) && len(candidate) > len(base) {
			base = candidate
		}
	}
	if base == "" {
		return nil, fmt.Errorf("unknown variable")
	}
	rest := strings.TrimPrefix(strings.TrimPrefix(name, base), "_")
	obj := map[string]any{}
	if base == "bucket_size" {
		obj["bucket"] = rest
	} else if err := parseVarFilters(base, rest, obj); err != nil {
		return nil, err
	}
	return allCounterParsers()[base](obj)
}

// unit_cost_food_13, building_cost_stone_23, tech_cost_gold_5: one resource of what something costs
func costCounter(name string) (counter, bool, error) {
	kinds := map[string]func(View, uint32) Cost{
		"unit_cost_":     func(v View, id uint32) Cost { return v.UnitCost(id) },
		"building_cost_": func(v View, id uint32) Cost { return v.BuildCost(id) },
		"tech_cost_":     func(v View, id uint32) Cost { return v.TechCost(id) },
	}
	for prefix, cost := range kinds {
		rest, found := strings.CutPrefix(name, prefix)
		if !found {
			continue
		}
		parts := strings.Split(rest, "_")
		if len(parts) != 2 {
			return nil, true, fmt.Errorf("expected %s<resource>_<id>", prefix)
		}
		category, err := parseResourceCategory(parts[0])
		if err != nil {
			return nil, true, err
		}
		id, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, true, fmt.Errorf("expected %s<resource>_<id>", prefix)
		}
		return func(v View, _ *Internals) float64 { return cost(v, uint32(id)).of(category) }, true, nil
	}
	return nil, false, nil
}

func parseVarFilters(base string, rest string, obj map[string]any) error {
	if rest == "" {
		return nil
	}
	tokens := strings.Split(rest, "_")
	ids := make([]any, 0)
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		if n, err := strconv.Atoi(t); err == nil {
			ids = append(ids, float64(n))
			continue
		}
		if _, ok := unitTypeNames[t]; ok {
			types, _ := obj["unit_types"].([]any)
			obj["unit_types"] = append(types, t)
			continue
		}
		if _, err := parseResourceCategory(t); err == nil {
			obj["category"] = t
			continue
		}
		switch t {
		case "within":
			if i+1 >= len(tokens) {
				return fmt.Errorf("\"within\" needs a number after it")
			}
			n, err := strconv.Atoi(tokens[i+1])
			if err != nil {
				return fmt.Errorf("\"within\" needs a number after it")
			}
			obj["within"] = float64(n)
			i++
			// within_N_of_<anchor> must come last: the anchor may itself contain underscores (bucket names)
			if i+2 < len(tokens) && tokens[i+1] == "of" {
				obj["from"] = strings.Join(tokens[i+2:], "_")
				return finishVarFilters(base, ids, obj)
			}
			continue
		case "complete", "damaged", "depleted":
			obj["state"] = t
			continue
		case "under", "without":
			if i+1 < len(tokens) && ((t == "under" && tokens[i+1] == "construction") || (t == "without" && tokens[i+1] == "builders")) {
				if t == "under" {
					obj["state"] = "under_construction"
				} else {
					obj["without_builders"] = true
				}
				i++
				continue
			}
		}
		return fmt.Errorf("unknown filter %q", t)
	}
	return finishVarFilters(base, ids, obj)
}

func finishVarFilters(base string, ids []any, obj map[string]any) error {
	if len(ids) > 0 {
		switch base {
		case "building_count", "foundation_count", "enemy_buildings_known":
			obj["building_ids"] = ids
		case "tech_researched":
			obj["tech_id"] = ids[0]
		default:
			obj["unit_ids"] = ids
		}
	}
	return nil
}

func allCounterParsers() map[string]func(obj map[string]any) (counter, error) {
	all := make(map[string]func(obj map[string]any) (counter, error), len(counterParsers)+2)
	for k, v := range counterParsers {
		all[k] = v
	}
	all["tech_researched"] = func(obj map[string]any) (counter, error) {
		tech_id := parseOptionalID(obj, "tech_id")
		if tech_id == nil {
			return nil, fmt.Errorf("tech_researched needs a tech id")
		}
		return func(v View, _ *Internals) float64 { return boolNumber(v.TechResearched(*tech_id)) }, nil
	}
	all["resource_available"] = func(obj map[string]any) (counter, error) {
		category, err := parseResourceCategory(obj["category"])
		if err != nil {
			return nil, err
		}
		condition := &conditionResourceAvailable{category: category}
		return func(v View, i *Internals) float64 { return boolNumber(condition.Evaluate(v, i)) }, nil
	}
	return all
}

func boolNumber(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func init() {
	// registered here rather than in the counterParsers literal, which would be an initialization cycle
	counterParsers["value"] = parseValueCounter
}

// "value" counter: compares any expression, so rules can branch on script variables
func parseValueCounter(obj map[string]any) (counter, error) {
	value, err := parseAmount(obj["value"])
	if err != nil {
		return nil, fmt.Errorf("\"value\" %v", err)
	}
	return func(v View, i *Internals) float64 { return value.float(v, i) }, nil
}
