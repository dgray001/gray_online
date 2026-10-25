package ai

import (
	"fmt"
	"maps"
)

const maxSelectSteps = 100000

func (i *Internals) takeSelectStep() bool {
	i.select_steps++
	if i.select_steps > maxSelectSteps {
		i.warnOnce(fmt.Sprintf("select_steps:%d", i.rule_index), fmt.Sprintf("ai: rule %d: selectors stopped after %d items", i.rule_index, maxSelectSteps))
		return false
	}
	return true
}

var aggregateNames = []string{"count", "sum", "min", "max"}

func distinctCount(items []selItem, field string, internals *Internals) float64 {
	values := map[float64]bool{}
	for _, item := range items {
		restore := internals.bindItem("item", item)
		previous := internals.eval_errors
		value, valid := item.field(field)
		restore()
		if !valid || internals.eval_errors != previous {
			return 0
		}
		values[value] = true
	}
	return float64(len(values))
}

func parseAggregate(name string, obj map[string]any) (counter, error) {
	of_raw, has_of := obj["of"]
	default_raw, has_default := obj["default"]
	distinct_raw, has_distinct := obj["distinct"]
	selector := maps.Clone(obj)
	delete(selector, "of")
	delete(selector, "default")
	delete(selector, "distinct")
	spec, err := parseSelectorSpec(selector, "item")
	if err != nil {
		return nil, fmt.Errorf("%s: %v", name, err)
	}
	distinct, valid := distinct_raw.(string)
	if has_distinct && (name != "count" || !valid || !hasItemField("select_"+spec.kind, distinct)) {
		return nil, fmt.Errorf("distinct needs a field of the counted selector kind")
	}
	var fallback *amount
	if has_default {
		if name != "min" && name != "max" {
			return nil, fmt.Errorf("%s takes no \"default\"; only min and max need one for an empty set", name)
		}
		parsed, parse_err := parseAmount(default_raw)
		if parse_err != nil {
			return nil, fmt.Errorf("%s default: %v", name, parse_err)
		}
		fallback = &parsed
	}
	var of amount
	switch {
	case name == "count" && has_of:
		return nil, fmt.Errorf("count takes no \"of\" expression")
	case name != "count" && !has_of:
		return nil, fmt.Errorf("%s needs an \"of\" expression", name)
	case name != "count":
		err = withLoopScopeExtra("item", "select_"+spec.kind, spec.letNames(), func() error {
			var parse_err error
			of, parse_err = parseAmount(of_raw)
			return parse_err
		})
		if err != nil {
			return nil, fmt.Errorf("%s of: %v", name, err)
		}
	}
	return func(view View, internals *Internals) float64 {
		return evaluateAggregate(name, spec, of, fallback, distinct, view, internals)
	}, nil
}

func evaluateAggregate(name string, spec *selectorSpec, of amount, fallback *amount, distinct string, view View, internals *Internals) float64 {
	cen := internals.select_census
	if cen == nil {
		cen = newCensus(view, internals)
	}
	items := spec.resolve(view, internals, cen)
	if name == "count" {
		if distinct != "" {
			return distinctCount(items, distinct, internals)
		}
		return float64(len(items))
	}
	if len(items) == 0 && fallback != nil {
		return fallback.float(view, internals)
	}
	total, best := 0.0, 0.0
	for n, item := range items {
		restore := internals.bindItem("item", item)
		value := of.float(view, internals)
		restore()
		total += value
		if n == 0 || (name == "min" && value < best) || (name == "max" && value > best) {
			best = value
		}
	}
	if name == "sum" {
		return total
	}
	return best
}
