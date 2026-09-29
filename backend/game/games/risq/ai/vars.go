package ai

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// setVarAction stores a number for later rules to read with var(name): in the per-turn map (cleared
// every turn) or, with "persist", in the script's global map that survives between turns.
type setVarAction struct {
	name    string
	value   amount
	persist bool
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
		var err error
		c, err = builtinCounter(name)
		if err != nil {
			i.warnOnce("var:"+name, fmt.Sprintf("ai var(%s): %v; using 0", name, err))
		}
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

// Built-in var() names mirror the count conditions, with their filters spelled out after underscores:
// population_13, population_infantry, idle_units_1, enemy_units_visible_13_within_2, building_count_23_complete,
// foundation_count_2_without_builders, resource_food, resource_remaining_food_within_1, bucket_size_<bucket>,
// tech_researched_5, resource_available_gold, plus the plain expression variables (turn, food, population_limit, ...)
func builtinCounter(name string) (counter, error) {
	for _, v := range expressionVarNames {
		if name == v {
			return func(view View, internals *Internals) float64 { return expressionVars(view, internals)[v] }, nil
		}
	}
	base := ""
	for candidate := range allCounterParsers() {
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
	if len(ids) > 0 {
		switch base {
		case "building_count", "foundation_count":
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
